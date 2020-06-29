package commons

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

func NewAppV1DaemonSet(name, namespace string, ownerRef *metav1.OwnerReference, labels map[string]string) *appsv1.DaemonSet {
	daemonSet := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.DaemonSet{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	if ownerRef != nil {
		daemonSet.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	}
	return daemonSet
}
