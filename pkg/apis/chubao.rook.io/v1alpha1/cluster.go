package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultServerImage     = "chubaofs/cfs-server:0.0.1"
	defaultDataDirHostPath = "/var/lib/chubao"
	defaultLogDirHostPath  = "/var/log/chubao"

	// Master
	masterDefaultReplicas            = 4
	masterDefaultLogLevel            = "error"
	masterDefaultRetainLogs          = 2000
	masterDefaultPort                = 17110
	masterDefaultProf                = 17120
	masterDefaultExporterPort        = 17150
	masterDefaultMetaNodeReservedMem = 67108864

	// MetaNode
	metaNodeDefaultLogLevel          = "error"
	metaNodeDefaultPort              = 17210
	metaNodeDefaultProf              = 17220
	metaNodeDefaultRaftHeartbeatPort = 17230
	metaNodeDefaultRaftReplicaPort   = 17240
	metaNodeDefaultExporterPort      = 17250
	metaNodeDefaultTotalMem          = 2147483648

	// DataNode
	dataNodeDefaultLogLevel          = "error"
	dataNodeDefaultPort              = 17310
	dataNodeDefaultProf              = 17320
	dataNodeDefaultRaftHeartbeatPort = 17330
	dataNodeDefaultRaftReplicaPort   = 17340
	dataNodeDefaultExporterPort      = 17350

	// Consul
	consulDefaultPort  = 8500
	consulDefaultImage = "consul:1.6.1"

	// CSI
	csiDefaultKubeletPath      = "/var/lib/kubelet"
	csiDefaultDriverName       = "csi.chubaofs.com"
	csiDefaultChubaoFSImage    = "chubaofs/cfs-csi-driver:2.2.1.110.0"
	csiDefaultProvisionerImage = "quay.io/k8scsi/csi-provisioner:v1.6.0"
	csiDefaultAttacherImage    = "quay.io/k8scsi/csi-attacher:v2.0.0"
	csiDefaultRegisterImage    = "quay.io/k8scsi/csi-node-driver-registrar:v1.3.0"
)

func SetDefault(cluster *ChubaoCluster) {
	image := cluster.Spec.Image
	if len(image) == 0 {
		cluster.Spec.Image = defaultServerImage
	}

	imagePullPolicy := cluster.Spec.ImagePullPolicy
	if len(imagePullPolicy) == 0 {
		cluster.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	}

	dataDirHostPath := cluster.Spec.DataDirHostPath
	if len(dataDirHostPath) == 0 {
		cluster.Spec.DataDirHostPath = defaultDataDirHostPath
	}

	logDirHostPath := cluster.Spec.LogDirHostPath
	if len(logDirHostPath) == 0 {
		cluster.Spec.LogDirHostPath = defaultLogDirHostPath
	}

	cleanupPolicy := cluster.Spec.CleanupPolicy
	if len(cleanupPolicy) == 0 {
		cluster.Spec.CleanupPolicy = CleanupPolicyNone
	}

	SetMasterDefault(&(cluster.Spec.Master))
	SetMetaNodeDefault(&(cluster.Spec.MetaNode))
	SetDataNodeDefault(&(cluster.Spec.DataNode))
	SetConsulDefault(&(cluster.Spec.Consul))
	SetProvisioner(&(cluster.Spec.Provisioner))
}

func SetProvisioner(p *ProvisionerSpec) {
	kubeletPath := p.KubeletPath
	if len(kubeletPath) == 0 {
		p.KubeletPath = csiDefaultKubeletPath
	}

	driverName := p.DriverName
	if len(driverName) == 0 {
		p.DriverName = csiDefaultDriverName
	}

	csiChubaoFSImage := p.CSIChubaoFS.Image
	if len(csiChubaoFSImage) == 0 {
		p.CSIChubaoFS.Image = csiDefaultChubaoFSImage
	}

	csiProvisionerImage := p.CSIProvisioner.Image
	if len(csiProvisionerImage) == 0 {
		p.CSIProvisioner.Image = csiDefaultProvisionerImage
	}

	csiAttacherImage := p.CSIAttacher.Image
	if len(csiAttacherImage) == 0 {
		p.CSIAttacher.Image = csiDefaultAttacherImage
	}

	csiRegisterImage := p.CSIRegister.Image
	if len(csiRegisterImage) == 0 {
		p.CSIRegister.Image = csiDefaultRegisterImage
	}
}

func SetConsulDefault(c *ConsulSpec) {
	image := c.Image
	if len(image) == 0 {
		c.Image = consulDefaultImage
	}

	port := c.Port
	if port == 0 {
		c.Port = consulDefaultPort
	}
}

func SetDataNodeDefault(dn *DataNodeSpec) {
	logLevel := dn.LogLevel
	if len(logLevel) == 0 {
		dn.LogLevel = dataNodeDefaultLogLevel
	}

	port := dn.Port
	if port == 0 {
		dn.Port = dataNodeDefaultPort
	}

	prof := dn.Prof
	if prof == 0 {
		dn.Prof = dataNodeDefaultProf
	}

	raftHeartbeatPort := dn.RaftHeartbeatPort
	if raftHeartbeatPort == 0 {
		dn.RaftHeartbeatPort = dataNodeDefaultRaftHeartbeatPort
	}

	raftReplicaPort := dn.RaftReplicaPort
	if raftReplicaPort == 0 {
		dn.RaftReplicaPort = dataNodeDefaultRaftReplicaPort
	}

	exporterPort := dn.ExporterPort
	if exporterPort == 0 {
		dn.ExporterPort = dataNodeDefaultExporterPort
	}
}

func SetMetaNodeDefault(mn *MetaNodeSpec) {
	logLevel := mn.LogLevel
	if len(logLevel) == 0 {
		mn.LogLevel = metaNodeDefaultLogLevel
	}

	port := mn.Port
	if port == 0 {
		mn.Port = metaNodeDefaultPort
	}

	prof := mn.Prof
	if prof == 0 {
		mn.Prof = metaNodeDefaultProf
	}

	raftHeartbeatPort := mn.RaftHeartbeatPort
	if raftHeartbeatPort == 0 {
		mn.RaftHeartbeatPort = metaNodeDefaultRaftHeartbeatPort
	}

	raftReplicaPort := mn.RaftReplicaPort
	if raftReplicaPort == 0 {
		mn.RaftReplicaPort = metaNodeDefaultRaftReplicaPort
	}

	exporterPort := mn.ExporterPort
	if exporterPort == 0 {
		mn.ExporterPort = metaNodeDefaultExporterPort
	}

	totalMem := mn.TotalMem
	if totalMem == 0 {
		mn.TotalMem = metaNodeDefaultTotalMem
	}
}

func SetMasterDefault(m *MasterSpec) {
	replicas := m.Replicas
	if replicas == 0 {
		m.Replicas = masterDefaultReplicas
	}

	logLevel := m.LogLevel
	if len(logLevel) == 0 {
		m.LogLevel = masterDefaultLogLevel
	}

	retainLogs := m.RetainLogs
	if retainLogs == 0 {
		m.RetainLogs = masterDefaultRetainLogs
	}

	port := m.Port
	if port == 0 {
		m.Port = masterDefaultPort
	}

	prof := m.Prof
	if prof == 0 {
		m.Prof = masterDefaultProf
	}

	exporterPort := m.ExporterPort
	if exporterPort == 0 {
		m.ExporterPort = masterDefaultExporterPort
	}

	metanodeReservedMem := m.MetaNodeReservedMem
	if metanodeReservedMem == 0 {
		m.MetaNodeReservedMem = masterDefaultMetaNodeReservedMem
	}
}
