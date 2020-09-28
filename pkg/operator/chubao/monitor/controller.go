package monitor

import (
	"context"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/monitor/grafana"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
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
	controllerName = "chubao-monitor-controller"
)

// Add adds a new Controller based on nodedrain.ReconcileNode and registers the relevant watches and handlers
func Add(mgr manager.Manager, context commons.Context) error {
	return add(mgr, newReconciler(mgr, context))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, context commons.Context) reconcile.Reconciler {
	return &ReconcileChubaoMonitor{
		Context: context,
	}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return errors.Wrapf(err, "failed to create a new controller %q", controllerName)
	}

	logger.Info("monitor controller successfully started")

	// Watch for changes to the nodes
	specChangePredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newMonitor, ok := e.ObjectNew.(*chubaoapi.ChubaoMonitor)
			if !ok {
				return true
			}
			oldMonitor, ok := e.ObjectOld.(*chubaoapi.ChubaoMonitor)
			if !ok {
				return true
			}

			return !reflect.DeepEqual(newMonitor.Spec, oldMonitor.Spec)
		},
	}

	logger.Debugf("watch for changes to the chubaomonitor")
	err = c.Watch(&source.Kind{Type: &chubaoapi.ChubaoMonitor{}}, &handler.EnqueueRequestForObject{}, specChangePredicate)
	if err != nil {
		return errors.Wrap(err, "failed to watch for monitor changes")
	}

	return nil
}

type ReconcileChubaoMonitor struct {
	commons.Context
}

func (r *ReconcileChubaoMonitor) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	monitor := &chubaoapi.ChubaoMonitor{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, monitor)
	if err != nil {
		// The Monitor resource may no longer exist, in which case we stop processing.
		if apierrors.IsNotFound(err) {
			logger.Errorf("monitor '%s' in work queue no longer exists\n", request.String())
			return reconcile.Result{}, nil
		}
		logger.Errorf("Unexpected error while getting monitor object: %s\n", err)
		return reconcile.Result{}, nil
	}

	logger.Infof("handling monitor object: %s", request.String())
	if !monitor.DeletionTimestamp.IsZero() {
		err = r.deleteMonitor(monitor)
		return reconcile.Result{}, err
	}

	// This is a new ChubaoMonitor
	err = r.createMonitor(monitor)
	return reconcile.Result{}, err
}

func (r *ReconcileChubaoMonitor) deleteMonitor(monitor *chubaoapi.ChubaoMonitor) error {
	logger.Infof("deleteCluster: %v\n", monitor)
	return nil
}

func (r *ReconcileChubaoMonitor) createMonitor(monitor *chubaoapi.ChubaoMonitor) error {

	ownerRef := newMonitorOwnerRef(monitor)
	err := CreateNewConfigmap(monitor)
	if err != nil {
		//		r.Recorder.Eventf(monitor, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateMonitorFailed, monitorKey)
		return errors.Wrap(err, "failed to create configmap")
	}
	logger.Infof("create configmap successfully")

	c := prometheus.New(r.ClientSet, r.Recorder, monitor, ownerRef)
	if err := c.Deploy(); err != nil {
		return errors.Wrap(err, "failed to deploy prometheus")
	}

	m := grafana.New(r.ClientSet, r.Recorder, monitor, ownerRef)
	if err := m.Deploy(); err != nil {
		return errors.Wrap(err, "failed to monitor grafana")
	}

	return nil
}
