package metanode

import (
	"fmt"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/cluster/master"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

const (
	instanceName = "rook-cfs-metanode"
)

const (
	// message
	MessageMetaNodeCreated = "MetaNode[%s] DaemonSet created"

	// error message
	MessageMetaNodeFailed = "Failed to create MetaNode[%s] DaemonSet"
)

type MetaNode struct {
	chubaoapi.MetaNodeSpec
	namespace   string
	name        string
	cluster     *chubaoapi.ChubaoCluster
	clusterSpec chubaoapi.ClusterSpec
	clientSet   kubernetes.Interface
	ownerRef    metav1.OwnerReference
	recorder    record.EventRecorder
}

func New(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *MetaNode {
	clusterSpec := clusterObj.Spec
	metaNodeObj := clusterSpec.MetaNode
	return &MetaNode{
		MetaNodeSpec: metaNodeObj,
		namespace:    clusterObj.Namespace,
		name:         clusterObj.Name,
		clientSet:    clientSet,
		recorder:     recorder,
		cluster:      clusterObj,
		clusterSpec:  clusterSpec,
		ownerRef:     ownerRef,
	}
}

func (mn *MetaNode) Deploy() error {
	labels := metaNodeLabels(mn.name)
	clientSet := mn.clientSet
	daemonSet := mn.newMetaNodeDaemonSet(labels)
	metaNodeKey := fmt.Sprintf("%s/%s", daemonSet.Namespace, daemonSet.Name)
	err := k8sutil.CreateDaemonSet(daemonSet.Name, daemonSet.Namespace, clientSet, daemonSet)
	if err != nil {
		mn.recorder.Eventf(mn.cluster, corev1.EventTypeNormal, constants.ErrCreateFailed, MessageMetaNodeFailed, metaNodeKey)
	}

	mn.recorder.Eventf(mn.cluster, corev1.EventTypeNormal, constants.SuccessCreated, MessageMetaNodeCreated, metaNodeKey)
	return err
}

func metaNodeLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentMetaNode, clusterName)
}

func (mn *MetaNode) newMetaNodeDaemonSet(labels map[string]string) *appsv1.DaemonSet {
	daemonSet := commons.NewAppV1DaemonSet(instanceName, mn.namespace, &mn.ownerRef, labels)
	daemonSet.Spec = appsv1.DaemonSetSpec{
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.OnDeleteDaemonSetStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: createPodSpec(mn),
		},
	}

	k8sutil.AddRookVersionLabelToDaemonSet(daemonSet)
	return daemonSet
}

func createPodSpec(mn *MetaNode) corev1.PodSpec {
	privileged := true
	pathType := corev1.HostPathDirectoryOrCreate
	pod := corev1.PodSpec{
		HostNetwork:       true,
		HostPID:           true,
		DNSPolicy:         corev1.DNSClusterFirstWithHostNet,
		PriorityClassName: mn.clusterSpec.PriorityClassName,
		ImagePullSecrets:  mn.cluster.Spec.ImagePullSecrets,
		InitContainers: []corev1.Container{
			{
				Name:            "check-service",
				Image:           mn.clusterSpec.Image,
				ImagePullPolicy: mn.clusterSpec.ImagePullPolicy,
				Command:         []string{"/cfs/bin/start.sh"},
				Args: []string{
					"check",
					master.GetMasterAddr(mn.cluster),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "metanode",
				Image:           mn.clusterSpec.Image,
				ImagePullPolicy: mn.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{
					"/bin/bash",
				},
				Args: []string{
					"-c",
					"/cfs/bin/start.sh metanode; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", mn.Port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", mn.Prof)},
					{Name: "CBFS_RAFT_HEARTBEAT_PORT", Value: fmt.Sprintf("%d", mn.RaftHeartbeatPort)},
					{Name: "CBFS_RAFT_REPLICA_PORT", Value: fmt.Sprintf("%d", mn.RaftReplicaPort)},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", mn.ExporterPort)},
					{Name: "CBFS_MASTER_ADDRS", Value: master.GetMasterAddr(mn.cluster)},
					{Name: "CBFS_LOG_LEVEL", Value: mn.LogLevel},
					{Name: "CBFS_CONSUL_ADDR", Value: consul.GetConsulUrl(mn.cluster)},
				},
				Ports: []corev1.ContainerPort{
					// Port Name must be no more than 15 characters
					{Name: "port", ContainerPort: mn.Port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: mn.Prof, Protocol: corev1.ProtocolTCP},
					{Name: "heartbeat-port", ContainerPort: mn.RaftHeartbeatPort, Protocol: corev1.ProtocolTCP},
					{Name: "replica-port", ContainerPort: mn.RaftReplicaPort, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: mn.ExporterPort, Protocol: corev1.ProtocolTCP},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: constants.VolumeNameForLogPath, MountPath: constants.DefaultLogPathInContainer},
					{Name: constants.VolumeNameForDataPath, MountPath: constants.DefaultDataPathInContainer},
				},
				Resources: mn.Resources,
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(int(mn.Port)),
						},
					},
					TimeoutSeconds: 10,
					PeriodSeconds:  30,
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name:         constants.VolumeNameForDataPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mn.clusterSpec.DataDirHostPath, Type: &pathType}},
			},
			{
				Name:         constants.VolumeNameForLogPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mn.clusterSpec.LogDirHostPath, Type: &pathType}},
			},
		},
	}

	placement := mn.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	} else {
		nodeSelector := make(map[string]string)
		nodeSelector[fmt.Sprintf("%s-%s", mn.namespace, constants.ComponentMetaNode)] = "enabled"
		pod.NodeSelector = nodeSelector
	}

	return pod
}
