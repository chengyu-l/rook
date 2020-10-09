package cluster

import (
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/commons"
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
