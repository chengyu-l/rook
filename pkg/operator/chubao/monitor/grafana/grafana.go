package grafana

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

const (
	InstanceName       = "grafana"
	DefaultServiceName = "grafana-service"
	DefaultPort        = 3000
	DefaultDomain      = "monitor.chubaofs.com"
	SecretName         = "grafana-user-account"
)

const (
	// message
	MessageGrafanaCreated        = "Grafana[%s] Deployment created"
	MessageGrafanaServiceCreated = "Grafana[%s] Service created"
	// error message
	MessageCreateGrafanaServiceFailed = "Failed to create Grafana[%s] Service"
	MessageCreateGrafanaFailed        = "Failed to create Grafana[%s] Deployment"
)

type Grafana struct {
	chubaoapi.GrafanaSpec
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
	ownerRef metav1.OwnerReference) *Grafana {
	return &Grafana{
		GrafanaSpec: monitorObj.Spec.Grafana,
		clientSet:   clientSet,
		recorder:    recorder,
		monitorObj:  monitorObj,
		ownerRef:    ownerRef,
		namespace:   monitorObj.Namespace,
		name:        monitorObj.Name,
	}
}

func (grafana *Grafana) Deploy() error {
	labels := grafanaLabels(grafana.name)
	clientSet := grafana.clientSet
	service := grafana.newGrafanaService(labels)
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if _, err := k8sutil.CreateOrUpdateService(clientSet, grafana.namespace, service); err != nil {
		grafana.recorder.Eventf(grafana.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateGrafanaServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreateGrafanaServiceFailed, serviceKey)
	}
	grafana.recorder.Eventf(grafana.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessageGrafanaServiceCreated, serviceKey)

	deployment := grafana.newGrafanaDeployment(labels)
	err := k8sutil.CreateDeployment(clientSet, deployment.Name, deployment.Namespace, deployment)
	grafanaKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if err != nil {
		grafana.recorder.Eventf(grafana.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateGrafanaFailed, grafanaKey)
		return errors.Wrapf(err, MessageCreateGrafanaFailed, serviceKey)
	}
	grafana.recorder.Eventf(grafana.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessageGrafanaCreated, grafanaKey)
	return err
}

func grafanaLabels(monitorName string) map[string]string {
	return commons.LabelsForMonitor(constants.ComponentGrafana, monitorName)
}

func (grafana *Grafana) newGrafanaService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(DefaultServiceName, grafana.namespace, &grafana.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: 3000, Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: labels,
	}
	return service
}

func (grafana *Grafana) newGrafanaDeployment(labels map[string]string) *appsv1.Deployment {
	deployment := commons.NewAppV1Deployment(InstanceName, grafana.namespace, &grafana.ownerRef, labels)
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
			Spec: createPodSpec(grafana),
		},
	}

	return deployment
}

func createPodSpec(grafana *Grafana) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		ImagePullSecrets: grafana.ImagePullSecrets,
		Containers: []corev1.Container{
			{
				Name:            "grafana",
				Image:           grafana.Image,
				ImagePullPolicy: grafana.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name: "port", ContainerPort: 3000, Protocol: corev1.ProtocolTCP,
					},
				},
				Resources:      grafana.Resources,
				Env:            createEnv(grafana),
				ReadinessProbe: createReadinessProbe(grafana),
				VolumeMounts:   createVolumeMounts(grafana),
			},
		},
		Volumes: createVolumes(grafana),
	}

	placement := grafana.monitorObj.Spec.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	}

	return pod
}

func createVolumes(grafana *Grafana) []corev1.Volume {
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

func createVolumeMounts(grafana *Grafana) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "monitor-config",
			MountPath: "/grafana/init.sh",
			SubPath:   "init.sh",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/grafana.ini",
			SubPath:   "grafana.ini",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/provisioning/dashboards/chubaofs.json",
			SubPath:   "chubaofs.json",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/provisioning/dashboards/dashboard.yml",
			SubPath:   "dashboard.yml",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/provisioning/datasources/datasource.yml",
			SubPath:   "datasource.yml",
		},
	}
}

func createEnv(grafana *Grafana) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "GF_AUTH_BASIC_ENABLED",
			Value: "true",
		},
		{
			Name:  "GF_AUTH_ANONYMOUS_ENABLED",
			Value: "false",
		},
		{
			Name: "GF_SECURITY_ADMIN_USER",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: SecretName},
					Key:                  "username",
				},
			},
		},
		{
			Name: "GF_SECURITY_ADMIN_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: SecretName},
					Key:                  "password",
				},
			},
		},
		{
			Name:  "GF_USERS_ALLOW_SIGN_UP",
			Value: "false",
		},
	}
}

func createReadinessProbe(grafana *Grafana) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/login",
				Port: utilintstr.FromInt(DefaultPort),
			},
		},
	}
}
