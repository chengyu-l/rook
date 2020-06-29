package cluster

import "time"

const (
	// defaultStatusCheckInterval is the interval to check the status of the chubao cluster
	defaultStatusCheckInterval = 60 * time.Second
)

type clusterStatusChecker struct {
}
