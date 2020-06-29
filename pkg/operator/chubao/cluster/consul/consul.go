package consul

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
)

const (
	instanceName             = "consul"
	defaultConsulServiceName = "consul-service"
)

const (
	// message
	MessageConsulCreated        = "Consul[%s] Deployment created"
	MessageConsulServiceCreated = "Consul[%s] Service created"

	// error message
	MessageCreateConsulServiceFailed = "Failed to create Consul[%s] Service"
	MessageCreateConsulFailed        = "Failed to create Consul[%s] Deployment"
)

func GetConsulUrl(clusterObj *chubaoapi.ChubaoCluster) string {
	if clusterObj == nil {
		return ""
	}

	return fmt.Sprintf("http://%s:%d",
		GetConsulServiceDomain(clusterObj.Namespace), clusterObj.Spec.Consul.Port)
}

func GetConsulServiceDomain(namespace string) string {
	return commons.GetServiceDomain(defaultConsulServiceName, namespace)
}

type Consul struct {
	chubaoapi.ConsulSpec
	cluster             *chubaoapi.ChubaoCluster
	context             *clusterd.Context
	kubeInformerFactory kubeinformers.SharedInformerFactory
	ownerRef            metav1.OwnerReference
	recorder            record.EventRecorder
	namespace           string
	name                string
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *Consul {
	consulObj := clusterObj.Spec.Consul
	return &Consul{
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		cluster:             clusterObj,
		ConsulSpec:          consulObj,
		ownerRef:            ownerRef,
		namespace:           clusterObj.Namespace,
		name:                clusterObj.Name,
	}
}

func (consul *Consul) Deploy() error {
	labels := consulLabels(consul.name)
	clientSet := consul.context.Clientset

	service := consul.newConsulService(labels)
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if _, err := k8sutil.CreateOrUpdateService(clientSet, consul.namespace, service); err != nil {
		consul.recorder.Eventf(consul.cluster, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateConsulServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreateConsulServiceFailed, serviceKey)
	}
	consul.recorder.Eventf(consul.cluster, corev1.EventTypeNormal, constants.SuccessCreated, MessageConsulServiceCreated, serviceKey)

	deployment := consul.newConsulDeployment(labels)
	err := k8sutil.CreateDeployment(clientSet, deployment.Name, deployment.Namespace, deployment)
	consulKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if err != nil {
		consul.recorder.Eventf(consul.cluster, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateConsulFailed, consulKey)
	}
	consul.recorder.Eventf(consul.cluster, corev1.EventTypeNormal, constants.SuccessCreated, MessageConsulCreated, consulKey)
	return nil
}

func consulLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentConsul, clusterName)
}

func (consul *Consul) newConsulService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(defaultConsulServiceName, consul.namespace, &consul.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: consul.Port, Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: labels,
	}
	return service
}

func (consul *Consul) newConsulDeployment(labels map[string]string) *appsv1.Deployment {
	deployment := commons.NewAppV1Deployment(instanceName, consul.namespace, &consul.ownerRef, labels)
	replicas := int32(1)
	deployment.Spec = appsv1.DeploymentSpec{
		Replicas: &replicas,
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
			Spec: createPodSpec(consul),
		},
	}

	return deployment
}

func createPodSpec(consul *Consul) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "consul-pod",
				Image:           consul.Image,
				ImagePullPolicy: consul.cluster.Spec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name: "port", ContainerPort: consul.Port, Protocol: corev1.ProtocolTCP,
					},
				},
				Resources: consul.Resources,
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(int(consul.Port)),
						},
					},
					TimeoutSeconds: 10,
					PeriodSeconds:  30,
				},
			},
		},
	}

	placement := consul.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	}

	return pod
}
