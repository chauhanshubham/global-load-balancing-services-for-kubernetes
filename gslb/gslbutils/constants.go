package gslbutils

import gdpalphav2 "github.com/vmware/global-load-balancing-services-for-kubernetes/internal/apis/amko/v1alpha2"

const (
	// GSLBKubePath is a temporary path to put the kubeconfig
	GSLBKubePath = "/tmp/gslb-kubeconfig"

	//AVISystem is the namespace where everything AVI related is created
	AVISystem = "avi-system"

	// Ingestion layer operations
	ObjectAdd    = "ADD"
	ObjectDelete = "DELETE"
	ObjectUpdate = "UPDATE"

	// Ingestion layer objects
	RouteType            = gdpalphav2.RouteObj
	IngressType          = gdpalphav2.IngressObj
	SvcType              = gdpalphav2.LBSvcObj
	GSFQDNType           = "GSFqdn"
	PassthroughRoute     = "passthrough"
	ThirdPartyMemberType = "ThirdPartyMember"
	HostRuleType         = "HostRule"

	// Refresh cycle for AVI cache in seconds
	DefaultRefreshInterval = 600

	// Store types
	AcceptedStore = "Accepted"
	RejectedStore = "Rejected"

	// Multi-cluster key lengths
	IngMultiClusterKeyLen = 6
	MultiClusterKeyLen    = 5
	GSFQDNKeyLen          = 3

	// Default values for Retry Operations
	SlowSyncTime      = 120
	SlowRetryQueue    = "SlowRetry"
	FastRetryQueue    = "FastRetry"
	DefaultRetryCount = 5

	// Identify objects created by AMKO
	AmkoUser = "amko-gslb"

	// Go routines in the rest layer
	NumRestWorkers = 8

	// Service Protocols
	ProtocolTCP = "TCP"
	ProtocolUDP = "UDP"

	// Health monitors
	SystemHealthMonitorTypeTCP   = "HEALTH_MONITOR_TCP"
	SystemHealthMonitorTypeUDP   = "HEALTH_MONITOR_UDP"
	SystemGslbHealthMonitorTCP   = "System-GSLB-TCP"
	SystemGslbHealthMonitorHTTP  = "HEALTH_MONITOR_HTTP"
	SystemGslbHealthMonitorHTTPS = "HEALTH_MONITOR_HTTPS"

	// default passthrough health monitor (TCP), to be used for all passthrough routes
	SystemGslbHealthMonitorPassthrough = "amko--passthrough-hm-tcp"

	// Ports for health monitoring
	DefaultTCPHealthMonitorPort   = "80"
	DefaultHTTPHealthMonitorPort  = 80
	DefaultHTTPSHealthMonitorPort = 443

	// Timeout for rest operations
	RestTimeoutSecs = 600

	// Env vars
	GslbLeader = "GSLB_CTRL_IP_ADDRESS"

	// HostRule status constants
	HostRuleAccepted = "Accepted"
	HostRuleRejected = "Rejected"
)
