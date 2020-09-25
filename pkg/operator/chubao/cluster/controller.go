package cluster

import (
	"context"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/cluster/datanode"
	"github.com/rook/rook/pkg/operator/chubao/cluster/master"
	"github.com/rook/rook/pkg/operator/chubao/cluster/metanode"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "chubao-cluster-controller"
)

// Add adds a new Controller based on nodedrain.ReconcileNode and registers the relevant watches and handlers
func Add(mgr manager.Manager, context commons.Context) error {
	return add(mgr, newReconciler(mgr, context))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, context commons.Context) reconcile.Reconciler {
	return &ReconcileChubaoCluster{
		Context: context,
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrapf(err, "failed to create a new %q", controllerName)
	}

	logger.Info("successfully started")

	// Watch for changes to the nodes
	specChangePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newCluster, ok := e.ObjectNew.(*chubaoapi.ChubaoCluster)
			if !ok {
				return true
			}
			oldCluster, ok := e.ObjectOld.(*chubaoapi.ChubaoCluster)
			if !ok {
				return true
			}

			return !reflect.DeepEqual(newCluster.Spec, oldCluster.Spec)
		},
	}

	logger.Debugf("watch for changes to the chubaocluster")
	err = c.Watch(&source.Kind{Type: &chubaoapi.ChubaoCluster{}}, &handler.EnqueueRequestForObject{}, specChangePredicate)
	if err != nil {
		return errors.Wrap(err, "failed to watch for node changes")
	}

	return nil
}

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
