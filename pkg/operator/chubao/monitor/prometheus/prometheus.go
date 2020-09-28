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
	utilintstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

//set below
var PrometheusServiceUrl string

const (
	// message
	MessagePrometheusCreated        = "Prometheus[%s] Deployment created"
	MessagePrometheusServiceCreated = "Prometheus[%s] Service created"

	// error message
	MessageCreatePrometheusServiceFailed = "Failed to create Prometheus[%s] Service"
	MessageCreatePrometheusFailed        = "Failed to create Prometheus[%s] Deployment"

	instanceName           = "prometheus"
	defaultPromServiceName = "prometheus-service"
)

type Prometheus struct {
	clientSet     kubernetes.Interface
	monitorObj    *chubaoapi.ChubaoMonitor
	prometheusObj chubaoapi.PrometheusSpec
	ownerRef      metav1.OwnerReference
	recorder      record.EventRecorder
	namespace     string
	name          string
}

func New(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) *Prometheus {
	promObj := monitorObj.Spec.Prometheus
	return &Prometheus{
		clientSet:     clientSet,
		recorder:      recorder,
		monitorObj:    monitorObj,
		prometheusObj: promObj,
		ownerRef:      ownerRef,
		namespace:     monitorObj.Namespace,
		name:          monitorObj.Name,
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

	fmt.Println("create prom service successfully")

	deployment := prom.newPrometheusDeployment(labels)
	err := k8sutil.CreateDeployment(clientSet, deployment.Name, deployment.Namespace, deployment)
	prometheusKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if err != nil {
		fmt.Println("create prom deployment successfully")
		prom.recorder.Eventf(prom.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreatePrometheusFailed, prometheusKey)
	}
	fmt.Println("create prom deployment successfully")
	prom.recorder.Eventf(prom.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessagePrometheusCreated, prometheusKey)
	return nil
}

func prometheusLabels(monitorname string) map[string]string {
	return commons.LabelsForMonitor(constants.ComponentPrometheus, monitorname)
}

func (prom *Prometheus) newPrometheusService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(defaultPromServiceName, prom.namespace, &prom.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: prom.prometheusObj.Port, TargetPort: utilintstr.IntOrString{IntVal: 9090}, Protocol: corev1.ProtocolTCP,
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
		Containers: []corev1.Container{
			{
				Name:            "prometheus-pod",
				Image:           prometheus.prometheusObj.Image,
				ImagePullPolicy: prometheus.prometheusObj.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name: "port", ContainerPort: prometheus.prometheusObj.Port, Protocol: corev1.ProtocolTCP,
					},
				},
				Resources: prometheus.prometheusObj.Resources,
				Env:       createEnv(prometheus),
				// If grafana pod show the err "back-off restarting failed container", run this command to keep the container running ang then run ./run.sh in the container to check real error.
				//          Command:        []string{"/bin/bash", "-ce", "tail -f /dev/null"},
				VolumeMounts: createVolumeMounts(prometheus),
			},
		},
		Volumes: createVolumes(prometheus),
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
		{
			Name: "prometheus-data",
			VolumeSource: corev1.VolumeSource{
				HostPath: prometheus.prometheusObj.HostPath,
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
		{
			Name:      "prometheus-data",
			MountPath: "/prometheus-data",
		},
	}
}

func createEnv(prometheus *Prometheus) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "CONSUL_ADDRESS",
			Value: prometheus.prometheusObj.ConsulUrl,
		},
		{
			Name:  "TZ",
			Value: " Asia/Shanghai",
		},
	}
}
