package monitor

import (
	"fmt"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/monitor/console"
	"github.com/rook/rook/pkg/operator/chubao/monitor/grafana"
	netv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

func createMonitorIngress(clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) error {
	ingress := &netv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitor",
			Namespace: monitorObj.Namespace,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": "nginx",
			},
		},
		Spec: netv1beta1.IngressSpec{
			Rules: []netv1beta1.IngressRule{
				{
					Host: grafana.DefaultDomain,
					IngressRuleValue: netv1beta1.IngressRuleValue{
						HTTP: &netv1beta1.HTTPIngressRuleValue{
							Paths: []netv1beta1.HTTPIngressPath{
								{
									Backend: netv1beta1.IngressBackend{
										ServiceName: grafana.DefaultServiceName,
										ServicePort: intstr.IntOrString{IntVal: grafana.DefaultPort},
									},
								},
							},
						},
					},
				},
				{
					Host: console.DefaultDomain,
					IngressRuleValue: netv1beta1.IngressRuleValue{
						HTTP: &netv1beta1.HTTPIngressRuleValue{
							Paths: []netv1beta1.HTTPIngressPath{
								{
									Backend: netv1beta1.IngressBackend{
										ServiceName: console.DefaultServiceName,
										ServicePort: intstr.IntOrString{IntVal: monitorObj.Spec.Console.Port},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if &ownerRef != nil {
		ingress.OwnerReferences = []metav1.OwnerReference{ownerRef}
	}

	_, err := clientSet.NetworkingV1beta1().Ingresses(monitorObj.Namespace).Create(ingress)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create ingress %s. %+v", ingress.Name, err)
		}

		_, err = clientSet.NetworkingV1beta1().Ingresses(monitorObj.Namespace).Update(ingress)
	}

	return err
}
