package cluster

import (
	"fmt"
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaorookio "github.com/rook/rook/pkg/apis/chubao.rook.io"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	informers "github.com/rook/rook/pkg/client/informers/externalversions/chubao.rook.io/v1alpha1"
	listers "github.com/rook/rook/pkg/client/listers/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/cluster/datanode"
	"github.com/rook/rook/pkg/operator/chubao/cluster/master"
	"github.com/rook/rook/pkg/operator/chubao/cluster/metanode"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/provisioner"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"reflect"
)

const (
	clusterQueueName = "chubao-cluster-queue"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubao-controller-cluster")

type ClusterEventHandler struct {
	cache.ResourceEventHandler
	context             *clusterd.Context
	clusterInformer     informers.ChubaoClusterInformer
	clusterLister       listers.ChubaoClusterLister
	kubeInformerFactory kubeinformers.SharedInformerFactory
	queue               workqueue.RateLimitingInterface
	recorder            record.EventRecorder
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	clusterInformer informers.ChubaoClusterInformer,
	recorder record.EventRecorder,
) *ClusterEventHandler {
	return &ClusterEventHandler{
		context:             context,
		clusterInformer:     clusterInformer,
		clusterLister:       clusterInformer.Lister(),
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		queue:               workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), clusterQueueName),
	}
}

// OnAdd calls AddFunc if it's not nil.
func (e *ClusterEventHandler) OnAdd(obj interface{}) {
	newCluster, ok := obj.(*chubaoapi.ChubaoCluster)
	if !ok {
		return
	}

	commons.EnqueueItem(e.queue, newCluster)
}

// OnUpdate calls UpdateFunc if it's not nil.
func (e *ClusterEventHandler) OnUpdate(oldObj, newObj interface{}) {
	newCluster, ok := newObj.(*chubaoapi.ChubaoCluster)
	if !ok {
		return
	}
	oldCluster, ok := oldObj.(*chubaoapi.ChubaoCluster)
	if !ok {
		return
	}

	// If the Spec is the same as the one in our cache, there aren't
	// any changes we are interested in.
	if reflect.DeepEqual(newCluster.Spec, oldCluster.Spec) {
		return
	}

	commons.EnqueueItem(e.queue, newCluster)
}

// OnDelete calls DeleteFunc if it's not nil.
func (e *ClusterEventHandler) OnDelete(obj interface{}) {
	newCluster, ok := obj.(*chubaoapi.ChubaoCluster)
	if !ok {
		return
	}

	key, _ := cache.MetaNamespaceKeyFunc(newCluster)
	logger.Infof("delete cluster object: %s", key)
}

func (e *ClusterEventHandler) RunWorker() {
	defer runtime.HandleCrash(func(err interface{}) {
		logger.Infof("ClusterEventHandler crash. err:%v", err)
	})

	for commons.ProcessNextWorkItem(e.queue, e.workFunc) {
	}
}

func (e *ClusterEventHandler) workFunc(key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s\n", key))
		return nil
	}

	lister := e.clusterInformer.Lister()
	// Get the Cluster resource with this namespace/name
	cluster, err := lister.ChubaoClusters(namespace).Get(name)
	if err != nil {
		// The Cluster resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("cluster '%s' in work queue no longer exists\n", key))
			return nil
		}
		return fmt.Errorf("Unexpected error while getting cluster object: %s\n", err)
	}

	logger.Infof("handling cluster object: %s", key)
	// DeepCopy here to ensure nobody messes with the cache.
	oldObj, newObj := cluster, cluster.DeepCopy()
	// If sync was successful and Status has changed, update the Cluster.
	if err = e.sync(newObj); err == nil && !reflect.DeepEqual(oldObj.Status, newObj.Status) {
		// TODO liuchengyu PatchClusterStatus
		//err = util.PatchClusterStatus(new, e.rookClient)
	}

	return err
}

func (e *ClusterEventHandler) sync(cluster *chubaoapi.ChubaoCluster) error {
	var err error
	chubaoapi.SetDefault(cluster)
	if cluster.DeletionTimestamp.IsZero() {
		// new
		err = e.createCluster(cluster)
	} else {
		// delete
		err = e.deleteCluster(cluster)
	}

	return err
}

func (e *ClusterEventHandler) deleteCluster(cluster *chubaoapi.ChubaoCluster) error {
	fmt.Printf("deleteCluster: %v\n", cluster)
	return nil
}

func (e *ClusterEventHandler) createCluster(cluster *chubaoapi.ChubaoCluster) error {
	ownerRef := newClusterOwnerRef(cluster)
	c := consul.New(e.context, e.kubeInformerFactory, e.recorder, cluster, ownerRef)
	if err := c.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy consul")
	}

	m := master.New(e.context, e.kubeInformerFactory, e.recorder, cluster, ownerRef)
	if err := m.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy master")
	}

	dn := datanode.New(e.context, e.kubeInformerFactory, e.recorder, cluster, ownerRef)
	if err := dn.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy datanode")
	}

	mn := metanode.New(e.context, e.kubeInformerFactory, e.recorder, cluster, ownerRef)
	if err := mn.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy metanode")
	}

	if commons.IsRookCSIEnableChubaoFS() {
		go startChubaoFSCSI(e.context, e.kubeInformerFactory, e.recorder, cluster, ownerRef)
	} else {
		logger.Infof("not deploy ChubaoFS CSI")
	}

	return nil
}

func startChubaoFSCSI(context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) {
	defer runtime.HandleCrash(func(err interface{}) {
		logger.Infof("deploy ChubaoFS crash. err:%v", err)
	})

	logger.Infof("deploy ChubaoFS CSI")
	provision := provisioner.New(context, kubeInformerFactory, recorder, clusterObj, ownerRef)
	if err := provision.Deploy(); err != nil {
		logger.Errorf("deploy ChubaoFS CSI fail. err:%v", err)
	}
}

func newClusterOwnerRef(own metav1.Object) metav1.OwnerReference {
	return *metav1.NewControllerRef(own, schema.GroupVersionKind{
		Group:   chubaorookio.CustomResourceGroupName,
		Version: chubaoapi.Version,
		Kind:    reflect.TypeOf(chubaoapi.ChubaoCluster{}).Name(),
	})
}
