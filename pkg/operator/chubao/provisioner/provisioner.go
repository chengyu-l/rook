package provisioner

import (
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

type Provisioner struct {
	chubaoapi.ProvisionerSpec
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
	ownerRef metav1.OwnerReference) *Provisioner {
	clusterSpec := clusterObj.Spec
	provisionerObj := clusterSpec.Provisioner
	return &Provisioner{
		ProvisionerSpec: provisionerObj,
		namespace:       clusterObj.Namespace,
		name:            clusterObj.Name,
		clientSet:       clientSet,
		recorder:        recorder,
		cluster:         clusterObj,
		clusterSpec:     clusterSpec,
		ownerRef:        ownerRef,
	}
}

func (p *Provisioner) Deploy() error {
	err := p.deployCSIController()
	if err != nil {
		return err
	}

	return p.deployCSINodeDriver()
}
