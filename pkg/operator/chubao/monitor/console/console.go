package console

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/chubao/monitor/grafana"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

const (
	instanceName       = "rook-cfs-console"
	DefaultServiceName = "console-service"
	DefaultDomain      = "console.chubaofs.com"
)

const (
	// message
	MessageConsoleCreated        = "Console[%s] Deployment created"
	MessageConsoleServiceCreated = "Console[%s] Service created"
	// error message
	MessageCreateConsoleServiceFailed = "Failed to create Console[%s] Service"
	MessageCreateConsoleFailed        = "Failed to create Console[%s] Deployment"
)

type Console struct {
	chubaoapi.ConsoleSpec
	clientSet      kubernetes.Interface
	recorder       record.EventRecorder
	ownerRef       metav1.OwnerReference
	monitorObj     *chubaoapi.ChubaoMonitor
	monitorObjSpec chubaoapi.MonitorSpec
	namespace      string
	name           string
}

func New(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) *Console {
	consoleObjSpec := monitorObj.Spec.Console
	return &Console{
		ConsoleSpec:    consoleObjSpec,
		monitorObj:     monitorObj,
		monitorObjSpec: monitorObj.Spec,
		clientSet:      clientSet,
		recorder:       recorder,
		ownerRef:       ownerRef,
		namespace:      monitorObj.Namespace,
		name:           monitorObj.Name,
	}
}

func (c *Console) Deploy() error {
	labels := objectStoreLabels(c.name)
	service := c.newConsoleService(labels)
	serviceKey := fmt.Sprintf("%s/%s", c.namespace, c.name)
	if _, err := k8sutil.CreateOrUpdateService(c.clientSet, c.namespace, service); err != nil {
		c.recorder.Eventf(c.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateConsoleServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreateConsoleServiceFailed, serviceKey)
	}
	c.recorder.Eventf(c.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessageConsoleServiceCreated, serviceKey)

	deployment := c.newConsoleDeployment(labels)
	err := k8sutil.CreateDeployment(c.clientSet, deployment.Name, deployment.Namespace, deployment)
	objectStoreKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if err != nil {
		c.recorder.Eventf(c.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateConsoleFailed, objectStoreKey)
		return errors.Wrapf(err, MessageCreateConsoleFailed, serviceKey)
	}
	c.recorder.Eventf(c.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessageConsoleCreated, objectStoreKey)
	return nil
}

func objectStoreLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentConsole, clusterName)
}

func (c *Console) newConsoleService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(DefaultServiceName, c.namespace, &c.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: c.Port, Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: labels,
	}
	return service
}

func (c *Console) newConsoleDeployment(labels map[string]string) *appsv1.Deployment {
	deployment := commons.NewAppV1Deployment(instanceName, c.namespace, &c.ownerRef, labels)
	deployment.Spec = appsv1.DeploymentSpec{
		Replicas: &c.Replicas,
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
			Spec: createPodSpec(c),
		},
	}

	return deployment
}

func createPodSpec(c *Console) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		ImagePullSecrets: c.ImagePullSecrets,
		Containers: []corev1.Container{
			{
				Name:            "console",
				Image:           c.Image,
				ImagePullPolicy: c.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{
					"/bin/bash",
				},
				Args: []string{
					"-c",
					"/cfs/bin/start.sh console; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_CLUSTER_NAME", Value: c.ClusterName},
					{Name: "CBFS_MASTER_ADDRS", Value: c.MasterAddr},
					{Name: "OBJECT_NODE_DOMAIN", Value: c.ObjectNodeDomain},
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", c.Port)},
					{Name: "CBFS_LOG_LEVEL", Value: c.LogLevel},
					{Name: "CBFS_GRAFANA_URL", Value: fmt.Sprintf("http://%s", grafana.DefaultDomain)},
					{Name: "CBFS_PROMETHEUS_ADDR", Value: prometheus.ServiceURLWithPort(c.monitorObj)},
					k8sutil.PodIPEnvVar("POD_IP"),
					k8sutil.NameEnvVar(),
				},
				Ports: []corev1.ContainerPort{
					{Name: "port", ContainerPort: c.Port, Protocol: corev1.ProtocolTCP},
				},
				Resources: c.Resources,
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(int(c.Port)),
						},
					},
					TimeoutSeconds: 10,
					PeriodSeconds:  30,
				},
			},
		},
	}

	placement := c.monitorObjSpec.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	}

	return pod
}
