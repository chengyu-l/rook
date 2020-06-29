package datanode

import (
	"fmt"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/cluster/master"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
	"strings"
)

const (
	instanceName = "datanode"
)

const (
	// message
	MessageDataNodeCreated = "DataNode[%s] DaemonSet created"

	// error message
	MessageDataNodeFailed = "Failed to create DataNode[%s] DaemonSet"
)

type DataNode struct {
	chubaoapi.DataNodeSpec
	namespace           string
	name                string
	cluster             *chubaoapi.ChubaoCluster
	clusterSpec         chubaoapi.ClusterSpec
	context             *clusterd.Context
	kubeInformerFactory kubeinformers.SharedInformerFactory
	ownerRef            metav1.OwnerReference
	recorder            record.EventRecorder
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *DataNode {
	clusterSpec := clusterObj.Spec
	dataNodeObj := clusterSpec.DataNode
	return &DataNode{
		DataNodeSpec:        dataNodeObj,
		namespace:           clusterObj.Namespace,
		name:                clusterObj.Name,
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		cluster:             clusterObj,
		clusterSpec:         clusterSpec,
		ownerRef:            ownerRef,
	}
}

func (dn *DataNode) Deploy() error {
	labels := dataNodeLabels(dn.name)
	clientSet := dn.context.Clientset
	daemonSet := dn.newDataNodeDaemonSet(labels)
	dataNodeKey := fmt.Sprintf("%s/%s", dn.namespace, dn.name)
	err := k8sutil.CreateDaemonSet(daemonSet.Name, daemonSet.Namespace, clientSet, daemonSet)
	if err != nil {
		dn.recorder.Eventf(dn.cluster, corev1.EventTypeNormal, constants.ErrCreateFailed, MessageDataNodeFailed, dataNodeKey)
	}

	dn.recorder.Eventf(dn.cluster, corev1.EventTypeNormal, constants.SuccessCreated, MessageDataNodeCreated, dataNodeKey)
	return err
}

func dataNodeLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentDataNode, clusterName)
}

func (dn *DataNode) newDataNodeDaemonSet(labels map[string]string) *appsv1.DaemonSet {
	daemonSet := commons.NewAppV1DaemonSet(instanceName, dn.namespace, &dn.ownerRef, labels)
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
			Spec: createPodSpec(dn),
		},
	}

	k8sutil.AddRookVersionLabelToDaemonSet(daemonSet)
	return daemonSet
}

func createPodSpec(dn *DataNode) corev1.PodSpec {
	privileged := true
	pathType := corev1.HostPathDirectoryOrCreate
	pod := &corev1.PodSpec{
		HostNetwork: true,
		HostPID:     true,
		DNSPolicy:   corev1.DNSClusterFirstWithHostNet,
		InitContainers: []corev1.Container{
			{
				Name:            "check-service",
				Image:           dn.clusterSpec.Image,
				ImagePullPolicy: dn.clusterSpec.ImagePullPolicy,
				Command:         []string{"/cfs/bin/start.sh"},
				Args: []string{
					"check",
					master.GetMasterAddr(dn.cluster),
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "datanode",
				Image:           dn.clusterSpec.Image,
				ImagePullPolicy: dn.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{"/bin/bash"},
				Args: []string{
					"-c",
					"/cfs/bin/start.sh datanode; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", dn.Port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", dn.Prof)},
					{Name: "CBFS_RAFT_HEARTBEAT_PORT", Value: fmt.Sprintf("%d", dn.RaftHeartbeatPort)},
					{Name: "CBFS_RAFT_REPLICA_PORT", Value: fmt.Sprintf("%d", dn.RaftReplicaPort)},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", dn.ExporterPort)},
					{Name: "CBFS_MASTER_ADDRS", Value: master.GetMasterAddr(dn.cluster)},
					{Name: "CBFS_LOG_LEVEL", Value: dn.LogLevel},
					{Name: "CBFS_CONSUL_ADDR", Value: consul.GetConsulUrl(dn.cluster)},
					{Name: "CBFS_DISKS", Value: strings.Join(dn.Disks, ",")},
					{Name: "CBFS_ZONE", Value: dn.Zone},
				},
				Ports: []corev1.ContainerPort{
					// Port Name must be no more than 15 characters
					{Name: "port", ContainerPort: dn.Port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: dn.Prof, Protocol: corev1.ProtocolTCP},
					{Name: "heartbeat-port", ContainerPort: dn.RaftHeartbeatPort, Protocol: corev1.ProtocolTCP},
					{Name: "replica-port", ContainerPort: dn.RaftReplicaPort, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: dn.ExporterPort, Protocol: corev1.ProtocolTCP},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: constants.VolumeNameForLogPath, MountPath: constants.DefaultLogPathInContainer},
					{Name: constants.VolumeNameForDataPath, MountPath: constants.DefaultDataPathInContainer},
				},
				Resources: dn.Resource,
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						TCPSocket: &corev1.TCPSocketAction{
							Port: intstr.FromInt(int(dn.Port)),
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
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: dn.clusterSpec.DataDirHostPath, Type: &pathType}},
			},
			{
				Name:         constants.VolumeNameForLogPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: dn.clusterSpec.LogDirHostPath, Type: &pathType}},
			},
		},
	}

	addDiskToVolume(dn, pod)

	placement := dn.Placement
	if placement != nil {
		placement.ApplyToPodSpec(pod)
	} else {
		nodeSelector := make(map[string]string)
		nodeSelector[fmt.Sprintf("%s-%s", dn.namespace, constants.ComponentDataNode)] = "enabled"
		pod.NodeSelector = nodeSelector
	}

	return *pod
}

func addDiskToVolume(dn *DataNode, pod *corev1.PodSpec) {
	pathType := corev1.HostPathDirectoryOrCreate
	for _, diskAndRetainSize := range dn.Disks {
		arr := strings.Split(diskAndRetainSize, ":")
		disk := arr[0]
		//name := fmt.Sprintf("disk-%d", i)
		vol := corev1.Volume{
			Name:         k8sutil.PathToVolumeName(disk),
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: disk, Type: &pathType}},
		}

		volMount := corev1.VolumeMount{Name: k8sutil.PathToVolumeName(disk), MountPath: disk}
		pod.Volumes = append(pod.Volumes, vol)
		pod.Containers[0].VolumeMounts = append(pod.Containers[0].VolumeMounts, volMount)
	}
}
