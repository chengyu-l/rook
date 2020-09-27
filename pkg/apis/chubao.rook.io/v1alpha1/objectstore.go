package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultObjectStoreImage = "chubaofs/cfs-server:0.0.1"
	defaultReplicas         = 3
	defaultLogLevel         = "error"
	defaultPort             = 17510
	defaultProf             = 17520
	defaultExportPort       = 17550
)

func SetObjectStoreDefault(objectStore *ChubaoObjectStore) {
	replicas := objectStore.Spec.Replicas
	if replicas == 0 {
		objectStore.Spec.Replicas = defaultReplicas
	}

	image := objectStore.Spec.Image
	if len(image) == 0 {
		objectStore.Spec.Image = defaultObjectStoreImage
	}

	imagePullPolicy := objectStore.Spec.ImagePullPolicy
	if len(imagePullPolicy) == 0 {
		objectStore.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	}

	logLevel := objectStore.Spec.LogLevel
	if len(logLevel) == 0 {
		objectStore.Spec.LogLevel = defaultLogLevel
	}

	port := objectStore.Spec.Port
	if port == 0 {
		objectStore.Spec.Port = defaultPort
	}

	prof := objectStore.Spec.Prof
	if prof == 0 {
		objectStore.Spec.Prof = defaultProf
	}

	exporterPort := objectStore.Spec.ExporterPort
	if exporterPort == 0 {
		objectStore.Spec.ExporterPort = defaultExportPort
	}

}
