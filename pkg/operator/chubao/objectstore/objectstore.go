package objectstore

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"reflect"
)

const (
	instanceName                  = "rook-cfs-objectstore"
	defaultObjectStoreServiceName = "objectstore-service"
)

const (
	// message
	MessageObjectStoreCreated        = "ObjectStore[%s] Deployment created"
	MessageObjectStoreServiceCreated = "ObjectStore[%s] Service created"

	// error message
	MessageCreateObjectStoreServiceFailed = "Failed to create ObjectStore[%s] Service"
	MessageCreateObjectStoreFailed        = "Failed to create ObjectStore[%s] Deployment"
)

type ObjectStore struct {
	chubaoapi.ObjectStoreSpec
	clientSet      kubernetes.Interface
	recorder       record.EventRecorder
	ownerRef       metav1.OwnerReference
	objectStoreObj *chubaoapi.ChubaoObjectStore
	namespace      string
	name           string
}

func NewObjectStore(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	objectStoreObj *chubaoapi.ChubaoObjectStore,
	ownerRef metav1.OwnerReference) *ObjectStore {
	objectStoreSpec := objectStoreObj.Spec
	return &ObjectStore{
		ObjectStoreSpec: objectStoreSpec,
		objectStoreObj:  objectStoreObj,
		clientSet:       clientSet,
		recorder:        recorder,
		ownerRef:        ownerRef,
		namespace:       objectStoreObj.Namespace,
		name:            objectStoreObj.Name,
	}
}

func (o *ObjectStore) Deploy() error {
	labels := objectStoreLabels(o.name)
	service := o.newObjectStoreService(labels)
	serviceKey := fmt.Sprintf("%s/%s", o.namespace, o.name)
	if _, err := k8sutil.CreateOrUpdateService(o.clientSet, o.namespace, service); err != nil {
		o.recorder.Eventf(o.objectStoreObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateObjectStoreServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreateObjectStoreServiceFailed, serviceKey)
	}
	o.recorder.Eventf(o.objectStoreObj, corev1.EventTypeNormal, constants.SuccessCreated, MessageObjectStoreServiceCreated, serviceKey)

	deployment := o.newObjectStoreDeployment(labels)
	err := k8sutil.CreateDeployment(o.clientSet, deployment.Name, deployment.Namespace, deployment)
	objectStoreKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if err != nil {
		o.recorder.Eventf(o.objectStoreObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateObjectStoreFailed, objectStoreKey)
	}
	o.recorder.Eventf(o.objectStoreObj, corev1.EventTypeNormal, constants.SuccessCreated, MessageObjectStoreCreated, objectStoreKey)
	return nil
}

func objectStoreLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentObjectStore, clusterName)
}

func (o *ObjectStore) newObjectStoreService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(defaultObjectStoreServiceName, o.namespace, &o.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: o.Port, Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: labels,
	}
	return service
}

func (o *ObjectStore) newObjectStoreDeployment(labels map[string]string) *appsv1.Deployment {
	deployment := commons.NewAppV1Deployment(instanceName, o.namespace, &o.ownerRef, labels)
	deployment.Spec = appsv1.DeploymentSpec{
		Replicas: &o.Replicas,
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: createPodSpec(o),
		},
	}

	return deployment
}

func createPodSpec(o *ObjectStore) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		ImagePullSecrets: o.ImagePullSecrets,
		Containers: []corev1.Container{
			{
				Name:            "objectstore",
				Image:           o.Image,
				ImagePullPolicy: o.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{
					"/bin/bash",
				},
				Args: []string{
					"-c",
					"/cfs/bin/start.sh objectnode; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_MASTER_ADDRS", Value: o.MasterAddr},
					{Name: "CBFS_DOMAINS", Value: o.Domains},
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", o.Port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", o.Prof)},
					{Name: "CBFS_LOG_LEVEL", Value: o.LogLevel},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", o.ExporterPort)},
					{Name: "CBFS_CONSUL_ADDR", Value: o.ConsulURL},
					k8sutil.PodIPEnvVar("POD_IP"),
					k8sutil.NameEnvVar(),
				},
				Ports: []corev1.ContainerPort{
					{Name: "port", ContainerPort: o.Port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: o.Prof, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: o.ExporterPort, Protocol: corev1.ProtocolTCP},
				},
				Resources: o.Resources,
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(int(o.Port)),
						},
					},
					TimeoutSeconds: 10,
					PeriodSeconds:  30,
				},
			},
		},
	}

	placement := o.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	}

	return pod
}

func newObjectStoreOwnerRef(own metav1.Object) metav1.OwnerReference {
	return *metav1.NewControllerRef(own, schema.GroupVersionKind{
		Group:   chubaoapi.CustomResourceGroup,
		Version: chubaoapi.Version,
		Kind:    reflect.TypeOf(chubaoapi.ChubaoObjectStore{}).Name(),
	})
}
