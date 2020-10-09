package cluster

import (
	"context"
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterQueueName = "chubao-cluster-queue"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubao-controller-cluster")

type ReconcileChubaoCluster struct {
	commons.Context
}

func (r *ReconcileChubaoCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	cluster := &chubaoapi.ChubaoCluster{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, cluster)
	if err != nil {
		// The Cluster resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			logger.Errorf("cluster '%s' in work queue no longer exists\n", request.String())
			return reconcile.Result{}, nil
		}
		logger.Errorf("Unexpected error while getting cluster object: %s\n", err)
		return reconcile.Result{}, nil
	}

	logger.Infof("handling cluster object: %s", request.String())
	if !cluster.DeletionTimestamp.IsZero() {
		err = r.deleteCluster(cluster)
		return reconcile.Result{}, err
	}

	// new
	chubaoapi.SetDefault(cluster)
	err = r.createCluster(cluster)
	return reconcile.Result{}, err
}

func (r *ReconcileChubaoCluster) deleteCluster(cluster *chubaoapi.ChubaoCluster) error {
	logger.Infof("deleteCluster: %v\n", cluster)
	return nil
}

func (r *ReconcileChubaoCluster) createCluster(cluster *chubaoapi.ChubaoCluster) error {
	ownerRef := newClusterOwnerRef(cluster)
	c := consul.New(r.ClientSet, r.Recorder, cluster, ownerRef)
	if err := c.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy consul")
	}

	m := master.New(r.ClientSet, r.Recorder, cluster, ownerRef)
	if err := m.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy master")
	}

	dn := datanode.New(r.ClientSet, r.Recorder, cluster, ownerRef)
	if err := dn.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy datanode")
	}

	mn := metanode.New(r.ClientSet, r.Recorder, cluster, ownerRef)
	if err := mn.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy metanode")
	}

	if commons.IsRookCSIEnableChubaoFS() {
		go startChubaoFSCSI(r.ClientSet, r.Recorder, cluster, ownerRef)
	} else {
		logger.Infof("not deploy ChubaoFS CSI")
	}

	return nil
}

func startChubaoFSCSI(clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) {
	defer runtime.HandleCrash(func(err interface{}) {
		logger.Infof("deploy ChubaoFS crash. err:%v", err)
	})

	logger.Infof("deploy ChubaoFS CSI")
	provision := provisioner.New(clientSet, recorder, clusterObj, ownerRef)
	if err := provision.Deploy(); err != nil {
		logger.Errorf("deploy ChubaoFS CSI fail. err:%v", err)
	}
}

func newClusterOwnerRef(own metav1.Object) metav1.OwnerReference {
	return *metav1.NewControllerRef(own, schema.GroupVersionKind{
		Group:   chubaoapi.CustomResourceGroup,
		Version: chubaoapi.Version,
		Kind:    reflect.TypeOf(chubaoapi.ChubaoCluster{}).Name(),
	})
}
