/*
Copyright 2018 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package constants

// kubernetes service dns domain
const (
	ServiceDomainSuffix = "svc.cluster.local"
)

// Generic Labels used on objects created by the operator.
const (
	AppName         = "rook-chubao"
	OperatorAppName = "rook-chubao-operator"

	ComponentLabel   = "chubao.rook.io/component"
	ManagedByLabel   = "chubao.rook.io/managed-by"
	ClusterNameLabel = "chubao.rook.io/cluster"
	MonitorNameLabel = "chubao.rook.io/monitor"

	ComponentConsul        = "consul"
	ComponentMaster        = "master"
	ComponentMetaNode      = "metanode"
	ComponentDataNode      = "datanode"
	ComponentClient        = "client"
	ComponentCSIController = "csi-controller"
	ComponentCSINode       = "csi-node"
	ComponentObjectStore   = "object-store"
	ComponentGrafana       = "grafana"
	ComponentPrometheus    = "prometheus"
)

const (
	SuccessCreated  = "Created"
	ErrCreateFailed = "ErrCreateFailed"
)

const (
	VolumeNameForLogPath       = "pod-log-path"
	VolumeNameForDataPath      = "pod-data-path"
	DefaultDataPathInContainer = "/cfs/data"
	DefaultLogPathInContainer  = "/cfs/logs"
)
