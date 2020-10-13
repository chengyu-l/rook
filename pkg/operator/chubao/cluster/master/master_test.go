package master

import (
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestMaster_getMasterPeers(t *testing.T) {
	m := &Master{
		MasterSpec: chubaoapi.MasterSpec{
			Replicas: 3,
			Port:     9000,
		},
		namespace: "test-namespace",
	}

	peers := m.getMasterPeers()
	assert.Equal(t, "1:master-0.master-service.test-namespace.svc.cluster.local:9000,2:master-1.master-service.test-namespace.svc.cluster.local:9000,3:master-2.master-service.test-namespace.svc.cluster.local:9000", peers)
}

func TestGetMasterAddrs(t *testing.T) {
	type args struct {
		clusterObj *chubaoapi.ChubaoCluster
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test-1",
			args: args{
				clusterObj: &chubaoapi.ChubaoCluster{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "rook-chubao",
					},
					Spec: chubaoapi.ClusterSpec{
						Master: chubaoapi.MasterSpec{
							Replicas: 3,
							Port:     17110,
						},
					},
				},
			},
			want: "master-0.master-service.rook-chubao.svc.cluster.local:17110,master-1.master-service.rook-chubao.svc.cluster.local:17110,master-2.master-service.rook-chubao.svc.cluster.local:17110",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMasterAddr(tt.args.clusterObj); got != tt.want {
				t.Errorf("GetMasterAddrs() = %v, want %v", got, tt.want)
			}
		})
	}
}
