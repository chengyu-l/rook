package objectstore

import (
	"context"
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
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
	controllerName = "chubao-objectstore-controller"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubao-objectstore-controller")

// Add adds a new Controller based on nodedrain.ReconcileNode and registers the relevant watches and handlers
func Add(mgr manager.Manager, context commons.Context) error {
	return add(mgr, newReconciler(mgr, context))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, context commons.Context) reconcile.Reconciler {
	return &ReconcileChubaoObjectStore{
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

	logger.Debugf("watch for changes to the ChubaoObjectStore")
	err = c.Watch(&source.Kind{Type: &chubaoapi.ChubaoObjectStore{}}, &handler.EnqueueRequestForObject{}, specChangePredicate)
	if err != nil {
		return errors.Wrap(err, "failed to watch for node changes")
	}

	return nil
}

type ReconcileChubaoObjectStore struct {
	commons.Context
}

func (r *ReconcileChubaoObjectStore) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	objectStore := &chubaoapi.ChubaoObjectStore{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, objectStore)
	if err != nil {
		// The Cluster resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			logger.Errorf("cluster '%s' in work queue no longer exists\n", request.String())
			return reconcile.Result{}, nil
		}
		logger.Errorf("Unexpected error while getting ChubaoObjectStore object: %s\n", err)
		return reconcile.Result{}, nil
	}

	logger.Infof("handling cluster object: %s", request.String())
	if !objectStore.DeletionTimestamp.IsZero() {
		err = r.deleteObjectStore(objectStore)
		if err != nil {
			logger.Errorf("deleteObjectStore key:%v err:%v", request.NamespacedName.String(), err)
		}
		return reconcile.Result{}, err
	}

	chubaoapi.SetObjectStoreDefault(objectStore)
	err = r.createObjectStore(objectStore)
	if err != nil {
		logger.Errorf("createObjectStore key:%v err:%v", request.NamespacedName.String(), err)
	}
	return reconcile.Result{}, err
}

func (r *ReconcileChubaoObjectStore) deleteObjectStore(objectStore *chubaoapi.ChubaoObjectStore) error {
	logger.Infof("deleteObjectStore: %v\n", objectStore)
	return nil
}

func (r *ReconcileChubaoObjectStore) createObjectStore(objectStore *chubaoapi.ChubaoObjectStore) error {
	ownerRef := newObjectStoreOwnerRef(objectStore)
	o := NewObjectStore(r.ClientSet, r.Recorder, objectStore, ownerRef)
	if err := o.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy ObjectStore")
	}

	return nil
}
