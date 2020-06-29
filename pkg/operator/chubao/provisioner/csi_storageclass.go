package provisioner

import (
	"fmt"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/cluster/master"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"reflect"
)

func (p *Provisioner) deployDefaultStorageClass() error {
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete
	sc := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(storagev1.StorageClass{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rook-cfs-sc",
			Namespace: p.namespace,
		},
		Provisioner:   p.DriverName,
		ReclaimPolicy: &reclaimPolicy,
		Parameters: map[string]string{
			"masterAddr": master.GetMasterServiceURL(p.cluster),
			"consulAddr": consul.GetConsulUrl(p.cluster),
		},
	}

	return CreateDeployment(p.context.Clientset, sc.Name, sc)
}

func CreateDeployment(clientset kubernetes.Interface, name string, sc *storagev1.StorageClass) error {
	_, err := clientset.StorageV1().StorageClasses().Create(sc)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = clientset.StorageV1().StorageClasses().Update(sc)
		}

		if err != nil {
			return fmt.Errorf("failed to start %s StorageClass: %+v\n%+v", name, err, sc)
		}
	}
	return err
}
