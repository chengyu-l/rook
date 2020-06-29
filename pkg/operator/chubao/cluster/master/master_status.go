package master

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"io/ioutil"
	"net/http"
)

type responseData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type ClusterInfo struct {
	Name       string `json:"Name"`
	LeaderAddr string `json:"LeaderAddr"`
}

type ClusterStatus struct {
	DataNodeStatInfo dataNodeStatInfo     `json:"dataNodeStatInfo,omitempty"`
	MetaNodeStatInfo metaNodeStatInfo     `json:"metaNodeStatInfo,omitempty"`
	ZoneStatInfo     map[string]*zoneStat `json:"ZoneStatInfo,omitempty"`
}

type dataNodeStatInfo struct {
	TotalGB     float32 `json:"TotalGB,omitempty"`
	UsedGB      float32 `json:"UsedGB,omitempty"`
	IncreasedGB float32 `json:"IncreasedGB,omitempty"`
	UsedRatio   float32 `json:"UsedRatio,omitempty"`
}

type metaNodeStatInfo struct {
	TotalGB     float32 `json:"TotalGB,omitempty"`
	UsedGB      float32 `json:"UsedGB,omitempty"`
	IncreasedGB float32 `json:"IncreasedGB,omitempty"`
	UsedRatio   float32 `json:"UsedRatio,omitempty"`
}

type zoneStat struct {
	DataNodeStat dataNodeStat `json:"dataNodeStat,omitempty"`
	MetaNodeStat metaNodeStat `json:"metaNodeStat,omitempty"`
}

type dataNodeStat struct {
	TotalGB       float32 `json:"TotalGB,omitempty"`
	UsedGB        float32 `json:"UsedGB,omitempty"`
	AvailGB       float32 `json:"AvailGB,omitempty"`
	UsedRatio     float32 `json:"UsedRatio,omitempty"`
	TotalNodes    int     `json:"TotalNodes,omitempty"`
	WritableNodes int     `json:"WritableNodes,omitempty"`
}

type metaNodeStat struct {
	TotalGB       float32 `json:"TotalGB,omitempty"`
	UsedGB        float32 `json:"UsedGB,omitempty"`
	AvailGB       float32 `json:"AvailGB,omitempty"`
	UsedRatio     float32 `json:"UsedRatio,omitempty"`
	TotalNodes    int     `json:"TotalNodes,omitempty"`
	WritableNodes int     `json:"WritableNodes,omitempty"`
}

func getMasterStatusURL(serviceName, namespace string, port int32) string {
	return fmt.Sprintf("http://%s:%d/cluster/stat", commons.GetServiceDomain(serviceName, namespace), port)
}

func (m *Master) QueryStatus() (*ClusterStatus, error) {
	addrs := GetMasterAddrs(m.cluster)
	err, bytes := queryClusterState(addrs[0])
	if err != nil {
		return nil, err
	}

	clusterStatus := &ClusterStatus{}
	responseData := &responseData{}
	responseData.Data = clusterStatus
	err = json.Unmarshal(bytes, responseData)
	return clusterStatus, err
}

func (cs *ClusterStatus) IsAvailable() bool {
	for _, value := range cs.ZoneStatInfo {
		if value.DataNodeStat.WritableNodes >= 3 && value.MetaNodeStat.WritableNodes >= 3 {
			return true
		}
	}

	return false
}

func GetClusterInfo(cluster *chubaoapi.ChubaoCluster) (*ClusterInfo, error) {
	addrs := GetMasterAddrs(cluster)
	err, bytes := queryClusterInfo(addrs[0])
	if err != nil {
		return nil, err
	}

	clusterInfo := &ClusterInfo{}
	responseData := &responseData{}
	responseData.Data = clusterInfo
	err = json.Unmarshal(bytes, responseData)
	return clusterInfo, err
}

func queryClusterInfo(addr string) (error, []byte) {
	url := fmt.Sprintf("http://%s/admin/getCluster", addr)
	return request(url)
}

func queryClusterState(addr string) (error, []byte) {
	url := fmt.Sprintf("http://%s/cluster/stat", addr)
	return request(url)
}

func request(url string) (error, []byte) {
	resp, err := http.Get(url)
	if err != nil {
		return errors.Errorf("request cluster fail. err:%v", err), nil
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("request cluster StatusCode[%d] not ok", resp.StatusCode), nil
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf("read stream from response of cluster fail. err:%v", err), nil
	}

	return nil, bytes
}
