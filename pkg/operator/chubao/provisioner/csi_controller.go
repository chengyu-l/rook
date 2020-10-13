package provisioner

import (
	"fmt"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiControllerInstanceName = "rook-cfs-csi-controller"
)

func (p *Provisioner) deployCSIController() error {
	labels := csiControllerLabels(p.name)
	deployment := commons.NewAppV1Deployment(csiControllerInstanceName, p.namespace, &p.ownerRef, labels)
	replicas := int32(1)
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: createCSIControllerPodSpec(p),
		},
	}

	clientSet := p.clientSet
	return k8sutil.CreateDeployment(clientSet, deployment.Name, deployment.Namespace, deployment)
}

func createCSIControllerPodSpec(p *Provisioner) corev1.PodSpec {
	privileged := true
	socketDirHostPathType := corev1.HostPathDirectoryOrCreate
	mountDirHostPathType := corev1.HostPathDirectory
	mountPropagation := corev1.MountPropagationBidirectional
	pod := corev1.PodSpec{
		PriorityClassName: p.cluster.Spec.PriorityClassName,
		ImagePullSecrets:  p.cluster.Spec.ImagePullSecrets,
		Containers: []corev1.Container{
			{
				Name:            "provisioner",
				Image:           p.CSIProvisioner.Image,
				ImagePullPolicy: p.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Args: []string{
					fmt.Sprintf("--provisioner=%s", p.DriverName),
					"--csi-address=$(ADDRESS)",
				},
				Env: []corev1.EnvVar{
					{Name: "TZ", Value: "Asia/Shanghai"},
					{Name: "ADDRESS", Value: "/csi/csi-controller.sock"},
				},
				Resources: p.CSIProvisioner.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "socket-dir", MountPath: "/csi"},
				},
			},
			{
				Name:            "attacher",
				Image:           p.CSIAttacher.Image,
				ImagePullPolicy: p.clusterSpec.ImagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Args: []string{
					"--csi-address=$(ADDRESS)",
				},
				Env: []corev1.EnvVar{
					{Name: "TZ", Value: "Asia/Shanghai"},
					{Name: "ADDRESS", Value: "/csi/csi-controller.sock"},
				},
				Resources: p.CSIAttacher.Resources,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "socket-dir", MountPath: "/csi"},
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
					{Name: "ADDRESS", Value: "/csi/csi-controller.sock"},
					{Name: "LOG_LEVEL", Value: "5"},
					{Name: "CSI_ENDPOINT", Value: "unix:///csi/csi-controller.sock"},
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
		},
	}

	placement := p.Placement
	if placement != nil {
		placement.ApplyToPodSpec(&pod)
	}

	return pod
}

func csiControllerLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentCSIController, clusterName)
}
