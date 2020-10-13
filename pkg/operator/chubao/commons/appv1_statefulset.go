package commons

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

func NewAppV1StatefulSet(name, namespace string, ownerRef *metav1.OwnerReference, labels map[string]string) *appsv1.StatefulSet {
	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.StatefulSet{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	if ownerRef != nil {
		statefulSet.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	}
	return statefulSet
}
