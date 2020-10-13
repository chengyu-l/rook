package master

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"strings"
)

const (
	instanceName             = "rook-cfs-master"
	defaultMasterServiceName = "master-service"
)

const (
	// message
	MessageMasterCreated        = "Master[%s] StatefulSet created"
	MessageMasterServiceCreated = "Master[%s] Service created"

	// error message
	MessageCreateMasterServiceFailed = "Failed to create Master[%s] Service"
	MessageCreateMasterFailed        = "Failed to create Master[%s] StatefulSet"
)

type Master struct {
	chubaoapi.MasterSpec
	clientSet   kubernetes.Interface
	recorder    record.EventRecorder
	ownerRef    metav1.OwnerReference
	cluster     *chubaoapi.ChubaoCluster
	clusterSpec chubaoapi.ClusterSpec
	namespace   string
	name        string
}

func New(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *Master {
	clusterSpec := clusterObj.Spec
	masterObj := clusterSpec.Master
	return &Master{
		MasterSpec:  masterObj,
		clientSet:   clientSet,
		recorder:    recorder,
		cluster:     clusterObj,
		clusterSpec: clusterSpec,
		ownerRef:    ownerRef,
		namespace:   clusterObj.Namespace,
		name:        clusterObj.Name,
	}
}

func (m *Master) Deploy() error {
	labels := masterLabels(m.cluster.Name)
	clientSet := m.clientSet

	service := m.newMasterService(labels)
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)
	if _, err := k8sutil.CreateOrUpdateService(clientSet, m.namespace, service); err != nil {
		m.recorder.Eventf(m.cluster, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateMasterServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreateMasterServiceFailed, serviceKey)
	}
	m.recorder.Eventf(m.cluster, corev1.EventTypeNormal, constants.SuccessCreated, MessageMasterServiceCreated, serviceKey)

	statefulSet := m.newMasterStatefulSet(labels)
	err := k8sutil.CreateStatefulSet(clientSet, statefulSet.Name, statefulSet.Namespace, statefulSet)
	masterKey := fmt.Sprintf("%s/%s", statefulSet.Namespace, statefulSet.Name)
	if err != nil {
		m.recorder.Eventf(m.cluster, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreateMasterFailed, masterKey)
	}

	m.recorder.Eventf(m.cluster, corev1.EventTypeNormal, constants.SuccessCreated, MessageMasterCreated, masterKey)
	return nil
}

func masterLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentMaster, clusterName)
}

func (m *Master) newMasterStatefulSet(labels map[string]string) *appsv1.StatefulSet {
	statefulSet := commons.NewAppV1StatefulSet(instanceName, m.namespace, &m.ownerRef, labels)
	statefulSet.Spec = appsv1.StatefulSetSpec{
		Replicas:            &m.Replicas,
		ServiceName:         defaultMasterServiceName,
		PodManagementPolicy: appsv1.OrderedReadyPodManagement,
		UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
			Type: appsv1.OnDeleteStatefulSetStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: createPodSpec(m),
		},
	}

	return statefulSet
}

func createPodSpec(m *Master) corev1.PodSpec {
	privileged := true
	pathType := corev1.HostPathDirectoryOrCreate
	pod := corev1.PodSpec{
		HostNetwork:       true,
		HostPID:           true,
		DNSPolicy:         corev1.DNSClusterFirstWithHostNet,
		PriorityClassName: m.clusterSpec.PriorityClassName,
		ImagePullSecrets:  m.cluster.Spec.ImagePullSecrets,
		Containers: []corev1.Container{
			{
				Name:            "master",
				Image:           m.clusterSpec.Image,
				ImagePullPolicy: m.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{
					"/bin/bash",
				},
				Args: []string{
					"-c",
					"/cfs/bin/start.sh master; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_CLUSTER_NAME", Value: m.name},
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", m.Port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", m.Prof)},
					{Name: "CBFS_MASTER_PEERS", Value: m.getMasterPeers()},
					{Name: "CBFS_RETAIN_LOGS", Value: fmt.Sprintf("%d", m.RetainLogs)},
					{Name: "CBFS_LOG_LEVEL", Value: m.LogLevel},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", m.ExporterPort)},
					{Name: "CBFS_CONSUL_ADDR", Value: consul.GetConsulUrl(m.cluster)},
					{Name: "CBFS_METANODE_RESERVED_MEM", Value: fmt.Sprintf("%d", m.MetaNodeReservedMem)},
					k8sutil.PodIPEnvVar("POD_IP"),
					k8sutil.NameEnvVar(),
				},
				Ports: []corev1.ContainerPort{
					// Port Name must be no more than 15 characters
					{Name: "port", ContainerPort: m.Port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: m.Prof, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: m.ExporterPort, Protocol: corev1.ProtocolTCP},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: constants.VolumeNameForLogPath, MountPath: constants.DefaultLogPathInContainer},
					{Name: constants.VolumeNameForDataPath, MountPath: constants.DefaultDataPathInContainer},
				},
				Resources: m.Resources,
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(int(m.Port)),
						},
					},
					TimeoutSeconds: 10,
					PeriodSeconds:  30,
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name:         constants.VolumeNameForLogPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: m.clusterSpec.LogDirHostPath, Type: &pathType}},
			},
			{
				Name:         constants.VolumeNameForDataPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: m.clusterSpec.DataDirHostPath, Type: &pathType}},
			},
		},
	}

	placement := m.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	} else {
		nodeSelector := make(map[string]string)
		nodeSelector[fmt.Sprintf("%s-%s", m.namespace, constants.ComponentMaster)] = "enabled"
		pod.NodeSelector = nodeSelector
	}

	return pod
}

func (m *Master) newMasterService(labels map[string]string) *corev1.Service {
	service := commons.NewCoreV1Service(defaultMasterServiceName, m.namespace, &m.ownerRef, labels)
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "port", Port: m.Port, Protocol: corev1.ProtocolTCP,
			},
		},
		Selector: labels,
	}
	return service
}

// 1:master-0.master-service.svc.cluster.local:17110,2:master-1.master-service.svc.cluster.local:17110,3:master-2.master-service.svc.cluster.local:17110
func (m *Master) getMasterPeers() string {
	urls := make([]string, 0)
	for i := 0; i < int(m.Replicas); i++ {
		urls = append(urls, fmt.Sprintf("%d:%s-%d.%s:%d", i+1, instanceName, i, GetMasterServiceDomain(m.namespace), m.Port))
	}

	return strings.Join(urls, ",")
}

// master-0.master-service.svc.cluster.local:17110,master-1.master-service.svc.cluster.local:17110,master-2.master-service.svc.cluster.local:17110
func GetMasterAddr(clusterObj *chubaoapi.ChubaoCluster) string {
	return strings.Join(GetMasterAddrs(clusterObj), ",")
}

// ["master-0.master-service.svc.cluster.local:17110","master-1.master-service.svc.cluster.local:17110","master-2.master-service.svc.cluster.local:17110"]
func GetMasterAddrs(clusterObj *chubaoapi.ChubaoCluster) []string {
	master := clusterObj.Spec.Master
	urls := make([]string, 0)
	for i := 0; i < int(master.Replicas); i++ {
		urls = append(urls, fmt.Sprintf("%s-%d.%s:%d", instanceName, i, GetMasterServiceDomain(clusterObj.Namespace), master.Port))
	}

	return urls
}

func GetMasterServiceDomain(namespace string) string {
	return commons.GetServiceDomain(defaultMasterServiceName, namespace)
}

func GetMasterServiceURL(clusterObj *chubaoapi.ChubaoCluster) string {
	master := clusterObj.Spec.Master
	return fmt.Sprintf("%s:%d", GetMasterServiceDomain(clusterObj.Namespace), master.Port)
}
