package monitor

import (
	"context"
	"fmt"
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaorookio "github.com/rook/rook/pkg/apis/chubao.rook.io"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/chubao/monitor/grafana"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubao-controller-monitor")

const (
	controllerName = "chubao-monitor-controller"

	// message
	MessageMonitorCreated = "Monitor[%s] created"
	// error message
	MessageCreateMonitorFailed = "Failed to create Monitor[%s]"
)

// Add adds a new Controller based on nodedrain.ReconcileNode and registers the relevant watches and handlers
func Add(mgr manager.Manager, context commons.Context) error {
	return add(mgr, newReconciler(mgr, context))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, context commons.Context) reconcile.Reconciler {
	return &ReconcileChubaoMonitor{
		Context: context,
		stopCh:  make(chan struct{}),
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
	stopCh chan struct{}
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
	monitor.Status.Grafana = chubaoapi.GrafanaStatusFailure
	monitor.Status.Prometheus = chubaoapi.PrometheusStatusFailure
	close(r.stopCh)
	logger.Infof("deleteCluster: %v\n", monitor)
	return nil
}

func (r *ReconcileChubaoMonitor) createMonitor(monitor *chubaoapi.ChubaoMonitor) error {

	ownerRef := newMonitorOwnerRef(monitor)
	monitorKey := fmt.Sprintf("%s/%s", monitor.Namespace, monitor.Name)

	monitor.Status.Prometheus = chubaoapi.PrometheusStatusUnknown
	monitor.Status.Grafana = chubaoapi.GrafanaStatusUnknown
	monitor.Status.Configmap = chubaoapi.ConfigmapStatusUnknown
	r.Client.Update(context.Background(), monitor)

	err := createNewConfigmap(monitor)
	if err != nil {
		r.Recorder.Eventf(monitor, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateMonitorFailed, monitorKey)
		return errors.Wrap(err, "failed to create configmap")
	}

	c := prometheus.New(r.ClientSet, r.Recorder, monitor, ownerRef)
	if err := c.Deploy(); err != nil {
		r.Recorder.Eventf(monitor, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateMonitorFailed, monitorKey)
		return errors.Wrap(err, "failed to deploy prometheus")
	}

	m := grafana.New(r.ClientSet, r.Recorder, monitor, ownerRef)
	if err := m.Deploy(); err != nil {
		r.Recorder.Eventf(monitor, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateMonitorFailed, monitorKey)
		return errors.Wrap(err, "failed to deploy grafana")
	}
	r.Recorder.Eventf(monitor, corev1.EventTypeNormal, constants.SuccessCreated, MessageMonitorCreated, monitorKey)

	r.startMonitoring(r.stopCh, monitor)

	return nil
}

func newMonitorOwnerRef(own metav1.Object) metav1.OwnerReference {
	return *metav1.NewControllerRef(own, schema.GroupVersionKind{
		Group:   chubaorookio.CustomResourceGroupName,
		Version: chubaoapi.Version,
		Kind:    reflect.TypeOf(chubaoapi.ChubaoMonitor{}).Name(),
	})
}
