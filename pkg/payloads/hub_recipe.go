package payloads

type K8sClusterOptions struct {
	ClusterName string `json:"clusterName"`
	// Ip address of the control plane node, if the is only one control plane node (No HA).
	// Required to use static IP addresses configuration
	ControlPlaneIpAddress string `json:"controlPlaneIpAddress,omitempty"`
	// Ip addresses of the control plane nodes, if there are multiple control plane nodes (HA).
	// Required to use static IP addresses configuration
	ControlPlaneIpAddresses []string `json:"controlPlaneIpAddresses,omitempty"`
	// Gateway IP address of the network.
	// Required to use static IP addresses configuration
	GatewayIpAddress string `json:"gatewayIpAddress,omitempty"`
	// Number of control plane nodes.
	// If more than 1, use ControlPlaneIpAddresses instead of ControlPlaneIpAddress
	ControlPlanePoolSize int `json:"controlPlanePoolSize"`
	// Kubernetes version
	K8sVersion string `json:"k8sVersion"`
	// Nameservers IP addresses.
	// Required to use static IP addresses configuration
	Nameservers []string `json:"nameservers,omitempty"`
	// Searches domains
	// Required to use static IP addresses configuration
	Searches []string `json:"searches,omitempty"`
	// Number of worker nodes
	NbNodes int `json:"nbNodes"`
	// Network UUID
	Network string `json:"network"`
	// Storage Repository UUID
	Sr string `json:"sr"`
	// SSH key to use to connect to the nodes
	SshKey string `json:"sshKey"`
	// VIP address of the cluster for HA configuration.
	// Required to use static IP addresses configuration
	VipAddress string `json:"vipAddress,omitempty"`
	// Worker nodes IP addresses.
	// Required to use static IP addresses configuration
	WorkerNodeIpAddresses []string `json:"workerNodeIpAddresses,omitempty"`
}
