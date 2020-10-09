package v1alpha1

const (
	defaultPrometheusImage  = "prom/prometheus:v2.13.1"
	defaultGrafanaImage     = "grafana/grafana:6.4.4"
	defaultConsoleImage     = "chubaofs/cfs-server:0.0.1"
	defaultConsoleReplicas  = 1
	defaultConsolePort      = 17610
	defaultClusterName      = "mycluster"
	defaultObjectNodeDomain = "console.chubaofs.com"
	defaultConsoleLogLevel  = "error"
)

func SetMonitorDefault(monitor *ChubaoMonitor) {
	prometheusImage := monitor.Spec.Prometheus.Image
	if len(prometheusImage) == 0 {
		monitor.Spec.Prometheus.Image = defaultPrometheusImage
	}

	grafanaImage := monitor.Spec.Grafana.Image
	if len(grafanaImage) == 0 {
		monitor.Spec.Grafana.Image = defaultGrafanaImage
	}

	consoleImage := monitor.Spec.Console.Image
	if len(consoleImage) == 0 {
		monitor.Spec.Console.Image = defaultConsoleImage
	}

	replicas := monitor.Spec.Console.Replicas
	if replicas == 0 {
		monitor.Spec.Console.Replicas = defaultConsoleReplicas
	}

	port := monitor.Spec.Console.Port
	if port == 0 {
		monitor.Spec.Console.Port = defaultConsolePort
	}

	clusterName := monitor.Spec.Console.ClusterName
	if len(clusterName) == 0 {
		monitor.Spec.Console.ClusterName = defaultClusterName
	}

	objectNodeDomain := monitor.Spec.Console.ObjectNodeDomain
	if len(objectNodeDomain) == 0 {
		monitor.Spec.Console.ObjectNodeDomain = defaultObjectNodeDomain
	}

	logLevel := monitor.Spec.Console.LogLevel
	if len(logLevel) == 0 {
		monitor.Spec.Console.LogLevel = defaultConsoleLogLevel
	}
}
