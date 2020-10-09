package monitor

import (
	"fmt"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"strings"

	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	GrafanaIniFilePath        = "/monitor/grafana/grafana.ini"
	GrafanaInitShFilePath     = "/monitor/grafana/init.sh"
	GrafanaCfsTemplFilePath   = "/monitor/grafana/provisioning/dashboards/chubaofs.json"
	GrafanaDashboardFilePath  = "/monitor/grafana/provisioning/dashboards/dashboard.yml"
	GrafanaDataSourceFilePath = "/monitor/grafana/provisioning/datasources/datasource.yml"
	PrometheusFilePath        = "/monitor/prometheus/prometheus.yml"
)

type MonitorConfigMap struct {
	clientSet  kubernetes.Interface
	recorder   record.EventRecorder
	ownerRef   metav1.OwnerReference
	monitorObj *chubaoapi.ChubaoMonitor
	namespace  string
	name       string
}

func New(
	clientSet kubernetes.Interface,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) *MonitorConfigMap {
	return &MonitorConfigMap{
		monitorObj: monitorObj,
		clientSet:  clientSet,
		recorder:   recorder,
		ownerRef:   ownerRef,
		namespace:  monitorObj.Namespace,
		name:       monitorObj.Name,
	}
}

func (m *MonitorConfigMap) Deploy() error {
	cfg := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitor-config",
			Namespace: m.namespace,
		},
		Data: make(map[string]string),
	}

	if &m.ownerRef != nil {
		cfg.OwnerReferences = []metav1.OwnerReference{m.ownerRef}
	}

	prometheusCfg, err := m.getPrometheusYml()
	if err != nil {
		return err
	}
	cfg.Data["prometheus.yml"] = prometheusCfg

	datasourceCfg, err := m.getDataSourceYml()
	if err != nil {
		return err
	}
	cfg.Data["datasource.yml"] = datasourceCfg

	grafanaIni, err := ioutil.ReadFile(GrafanaIniFilePath)
	if err != nil {
		return err
	}
	cfg.Data["grafana.ini"] = string(grafanaIni)

	grafanaInitSh, err := ioutil.ReadFile(GrafanaInitShFilePath)
	if err != nil {
		return err
	}
	cfg.Data["init.sh"] = string(grafanaInitSh)

	cfsTempl, err := ioutil.ReadFile(GrafanaCfsTemplFilePath)
	if err != nil {
		return err
	}
	cfg.Data["chubaofs.json"] = string(cfsTempl)

	dashboard, err := ioutil.ReadFile(GrafanaDashboardFilePath)
	if err != nil {
		return err
	}
	cfg.Data["dashboard.yml"] = string(dashboard)

	_, err = m.clientSet.CoreV1().ConfigMaps(m.namespace).Create(cfg)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create configmap %s. %+v", cfg.Name, err)
		}

		_, err = m.clientSet.CoreV1().ConfigMaps(m.namespace).Update(cfg)
		if err != nil {
			return fmt.Errorf("failed to update configmap %s. %+v", cfg.Name, err)
		}
	}

	return nil
}

func (m *MonitorConfigMap) getPrometheusYml() (string, error) {
	bytes, err := ioutil.ReadFile(PrometheusFilePath)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(string(bytes), "192.168.0.101:8500", m.monitorObj.Spec.Prometheus.ConsulUrl), nil
}

func (m *MonitorConfigMap) getDataSourceYml() (string, error) {
	bytes, err := ioutil.ReadFile(GrafanaDataSourceFilePath)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(string(bytes), "192.168.0.102:9090", prometheus.ServiceURLWithPort(m.monitorObj)), nil
}
