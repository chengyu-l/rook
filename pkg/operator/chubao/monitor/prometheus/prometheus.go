package prometheus

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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

const (
	// message
	MessagePrometheusCreated        = "Prometheus[%s] Deployment created"
	MessagePrometheusServiceCreated = "Prometheus[%s] Service created"

	// error message
	MessageCreatePrometheusServiceFailed = "Failed to create Prometheus[%s] Service"
	MessageCreatePrometheusFailed        = "Failed to create Prometheus[%s] Deployment"

	instanceName           = "prometheus"
	defaultPromServiceName = "prometheus-service"
	prometheusPort         = 9090
)

type Prometheus struct {
	chubaoapi.PrometheusSpec
	clientSet  kubernetes.Interface
	monitorObj *chubaoapi.ChubaoMonitor
	ownerRef   metav1.OwnerReference
	recorder   record.EventRecorder
	namespace  string
	name       string
}

func New(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) *Prometheus {
	return &Prometheus{
		PrometheusSpec: monitorObj.Spec.Prometheus,
		clientSet:      clientSet,
		recorder:       recorder,
		monitorObj:     monitorObj,
		ownerRef:       ownerRef,
		namespace:      monitorObj.Namespace,
		name:           monitorObj.Name,
	}
}

func (prom *Prometheus) Deploy() error {
	labels := prometheusLabels(prom.name)
	clientSet := prom.clientSet

	service := prom.newPrometheusService(labels)
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if _, err := k8sutil.CreateOrUpdateService(clientSet, prom.namespace, service); err != nil {
		prom.recorder.Eventf(prom.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreatePrometheusServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreatePrometheusServiceFailed, serviceKey)
	}
	prom.recorder.Eventf(prom.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessagePrometheusServiceCreated, serviceKey)

	deployment := prom.newPrometheusDeployment(labels)
	err := k8sutil.CreateDeployment(clientSet, deployment.Name, deployment.Namespace, deployment)
	prometheusKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if err != nil {
		prom.recorder.Eventf(prom.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreatePrometheusFailed, prometheusKey)
		return errors.Wrapf(err, MessageCreatePrometheusFailed, serviceKey)
	}

	prom.recorder.Eventf(prom.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessagePrometheusCreated, prometheusKey)
	return nil
}

func prometheusLabels(monitorName string) map[string]string {
	return commons.LabelsForMonitor(constants.ComponentPrometheus, monitorName)
}

func (prom *Prometheus) newPrometheusService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(defaultPromServiceName, prom.namespace, &prom.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: prometheusPort, Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: labels,
	}
	return service

}

func (prom *Prometheus) newPrometheusDeployment(labels map[string]string) *appsv1.Deployment {
	deployment := commons.NewAppV1Deployment(instanceName, prom.namespace, &prom.ownerRef, labels)
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
			Spec: createPodSpec(prom),
		},
	}

	return deployment

}

func createPodSpec(prometheus *Prometheus) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		ImagePullSecrets: prometheus.ImagePullSecrets,
		Containers: []corev1.Container{
			{
				Name:            "prometheus-pod",
				Image:           prometheus.Image,
				ImagePullPolicy: prometheus.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name: "port", ContainerPort: prometheusPort, Protocol: corev1.ProtocolTCP,
					},
				},
				Resources:    prometheus.Resources,
				Env:          createEnv(prometheus),
				VolumeMounts: createVolumeMounts(prometheus),
			},
		},
		Volumes: createVolumes(prometheus),
	}

	placement := prometheus.monitorObj.Spec.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	}
	return pod
}

func createVolumes(prometheus *Prometheus) []corev1.Volume {
	var defaultmode int32 = 0555

	return []corev1.Volume{
		{
			Name: "monitor-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "monitor-config",
					},
					DefaultMode: &defaultmode,
				},
			},
		},
	}
}

func createVolumeMounts(prometheus *Prometheus) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "monitor-config",
			MountPath: "/etc/prometheus/prometheus.yml",
			SubPath:   "prometheus.yml",
		},
	}
}

func createEnv(prometheus *Prometheus) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "TZ",
			Value: " Asia/Shanghai",
		},
	}
}

func ServiceURLWithPort(monitorObj *chubaoapi.ChubaoMonitor) string {
	return fmt.Sprintf("%s:%d", commons.GetServiceDomain(defaultPromServiceName, monitorObj.Namespace), prometheusPort)
}
