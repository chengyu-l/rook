package provisioner

import (
	"fmt"
	"github.com/prometheus/common/log"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiDriverInstanceName = "rook-cfs-csi-node"
)

func csiNodeLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentCSINode, clusterName)
}

func (p *Provisioner) deployCSINodeDriver() error {
	labels := csiNodeLabels(p.name)
	daemonSet := commons.NewAppV1DaemonSet(csiDriverInstanceName, p.namespace, &p.ownerRef, labels)
	daemonSet.Spec = appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.OnDeleteDaemonSetStrategyType,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: createCSINodePod(p),
		},
	}

	k8sutil.AddRookVersionLabelToDaemonSet(daemonSet)
	clientSet := p.context.Clientset
	return k8sutil.CreateDaemonSet(daemonSet.Name, daemonSet.Namespace, clientSet, daemonSet)
}

func createCSINodePod(p *Provisioner) corev1.PodSpec {
	privileged := true
	socketDirHostPathType := corev1.HostPathDirectoryOrCreate
	mountDirHostPathType := corev1.HostPathDirectory
	mountPropagation := corev1.MountPropagationBidirectional
	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "driver-registrar",
				Image:           p.CSIRegister.Image,
				ImagePullPolicy: p.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Args: []string{
					"--kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)",
					"--csi-address=$(ADDRESS)",
				},
				Env: []corev1.EnvVar{
					{Name: "TZ", Value: "Asia/Shanghai"},
					{Name: "ADDRESS", Value: "/csi/csi.sock"},
					{Name: "DRIVER_REG_SOCK_PATH", Value: fmt.Sprintf("%s/plugins/csi.chubaofs.com/csi.sock", p.KubeletPath)},
					{Name: "KUBE_NODE_NAME", ValueFrom: k8sutil.NameEnvVar().ValueFrom},
				},
				Resources: p.CSIProvisioner.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "socket-dir", MountPath: "/csi"},
					{Name: "registration-dir", MountPath: "/registration"},
				},
			},
			{
				Name:            "cfs-driver",
				Image:           p.CSIChubaoFS.Image,
				ImagePullPolicy: p.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Args: []string{
					"bash",
					"-c",
					"set -e; su -p -s /bin/bash -c \"/cfs/bin/start.sh &\"; su -p -s /bin/bash -c \"sleep 9999999d\"",
				},
				Env: []corev1.EnvVar{
					{Name: "TZ", Value: "Asia/Shanghai"},
					{Name: "LOG_LEVEL", Value: "5"},
					{Name: "CSI_ENDPOINT", Value: "unix:///csi/csi.sock"},
					{Name: "DRIVER_NAME", Value: p.DriverName},
					{Name: "KUBE_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"}}},
				},
				Resources: p.CSIAttacher.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "socket-dir", MountPath: "/csi"},
					{Name: "mountpoint-dir", MountPath: fmt.Sprintf("%s/pods", p.KubeletPath), MountPropagation: &mountPropagation},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: "socket-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: fmt.Sprintf("%s/plugins/csi.chubaofs.com", p.KubeletPath),
						Type: &socketDirHostPathType,
					},
				},
			},
			{
				Name: "mountpoint-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: fmt.Sprintf("%s/pods", p.KubeletPath),
						Type: &mountDirHostPathType,
					},
				},
			},
			{
				Name: "registration-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: fmt.Sprintf("%s/plugins_registry", p.KubeletPath),
						Type: &socketDirHostPathType,
					},
				},
			},
		},
	}

	placement := p.Placement
	if placement != nil {
		log.Errorf("Placement:%v", &placement)
		placement.ApplyToPodSpec(&pod)
	}

	return pod
}
