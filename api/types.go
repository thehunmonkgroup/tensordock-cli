package api

import "encoding/json"

type SecretSummary struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Secret struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type SecretCreateRequest struct {
	Data struct {
		Type       string                 `json:"type"`
		Attributes SecretCreateAttributes `json:"attributes"`
	} `json:"data"`
}

type SecretCreateAttributes struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type ResourceLimits struct {
	MaxVCPUs     int `json:"max_vcpus"`
	MaxRAMGB     int `json:"max_ram_gb"`
	MaxStorageGB int `json:"max_storage_gb"`
}

type Pricing struct {
	PerVCPUHR      float64 `json:"per_vcpu_hr"`
	PerGBRAMHR     float64 `json:"per_gb_ram_hr"`
	PerGBStorageHR float64 `json:"per_gb_storage_hr"`
}

type NetworkFeatures struct {
	DedicatedIPAvailable   bool `json:"dedicated_ip_available"`
	PortForwardingAvailble bool `json:"port_forwarding_available"`
	NetworkStorageAvailble bool `json:"network_storage_available"`
}

type LocationGPU struct {
	V0Name      string          `json:"v0Name"`
	DisplayName string          `json:"displayName"`
	MaxCount    int             `json:"max_count"`
	PricePerHR  float64         `json:"price_per_hr"`
	Resources   ResourceLimits  `json:"resources"`
	Pricing     Pricing         `json:"pricing"`
	Network     NetworkFeatures `json:"network_features"`
}

type Location struct {
	ID            string        `json:"id"`
	City          string        `json:"city"`
	StateProvince string        `json:"stateprovince"`
	Country       string        `json:"country"`
	Tier          int           `json:"tier"`
	GPUs          []LocationGPU `json:"gpus"`
}

type HostnodeGPU struct {
	V0Name         string  `json:"v0Name"`
	AvailableCount int     `json:"availableCount"`
	PricePerHR     float64 `json:"price_per_hr"`
}

type HostnodeLocation struct {
	UUID                   string `json:"uuid"`
	City                   string `json:"city"`
	StateProvince          string `json:"stateprovince"`
	Country                string `json:"country"`
	HasNetworkStorage      bool   `json:"has_network_storage"`
	NetworkSpeedGbps       int    `json:"network_speed_gbps"`
	NetworkSpeedUploadGbps int    `json:"network_speed_upload_gbps"`
	Organization           string `json:"organization"`
	OrganizationName       string `json:"organizationName"`
	Tier                   int    `json:"tier"`
}

type HostnodeAvailableResources struct {
	GPUs                 []HostnodeGPU `json:"gpus"`
	VCPUCount            int           `json:"vcpu_count,omitempty"`
	RAMGB                int           `json:"ram_gb,omitempty"`
	StorageGB            int           `json:"storage_gb,omitempty"`
	MaxVCPUsPerGPU       int           `json:"max_vcpus_per_gpu,omitempty"`
	MaxRAMPerGPU         int           `json:"max_ram_per_gpu,omitempty"`
	MaxVCPUs             int           `json:"max_vcpus,omitempty"`
	MaxRAMGB             int           `json:"max_ram_gb,omitempty"`
	MaxStorageGB         int           `json:"max_storage_gb,omitempty"`
	AvailablePorts       []int         `json:"available_ports,omitempty"`
	HasPublicIPAvailable bool          `json:"has_public_ip_available"`
}

type Hostnode struct {
	ID                 string                     `json:"id"`
	LocationID         string                     `json:"location_id"`
	Engine             string                     `json:"engine"`
	UptimePercentage   float64                    `json:"uptime_percentage"`
	AvailableResources HostnodeAvailableResources `json:"available_resources"`
	Pricing            Pricing                    `json:"pricing"`
	Location           HostnodeLocation           `json:"location"`
}

type InstanceListItem struct {
	Type       string                     `json:"type"`
	ID         string                     `json:"id"`
	Name       string                     `json:"name,omitempty"`
	Status     string                     `json:"status,omitempty"`
	Attributes InstanceListItemAttributes `json:"attributes"`
}

type InstanceListItemAttributes struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Region string `json:"region,omitempty"`
}

type PortForward struct {
	InternalPort int `json:"internal_port"`
	ExternalPort int `json:"external_port"`
}

type GPUCount struct {
	Count  int    `json:"count"`
	V0Name string `json:"v0Name,omitempty"`
}

type InstanceResources struct {
	VCPUCount int                 `json:"vcpu_count"`
	RAMGB     int                 `json:"ram_gb"`
	StorageGB int                 `json:"storage_gb"`
	GPUs      map[string]GPUCount `json:"gpus,omitempty"`
}

type Instance struct {
	Type         string            `json:"type"`
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	IPAddress    string            `json:"ipAddress"`
	PortForwards []PortForward     `json:"portForwards"`
	Resources    InstanceResources `json:"resources"`
	RateHourly   float64           `json:"rateHourly"`
}

type InstanceCreateRequest struct {
	Data struct {
		Type       string                   `json:"type"`
		Attributes InstanceCreateAttributes `json:"attributes"`
	} `json:"data"`
}

type InstanceCreateAttributes struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Image          string            `json:"image"`
	Resources      InstanceResources `json:"resources"`
	LocationID     string            `json:"location_id,omitempty"`
	HostnodeID     string            `json:"hostnode_id,omitempty"`
	UseDedicatedIP bool              `json:"useDedicatedIp,omitempty"`
	PortForwards   []PortForward     `json:"port_forwards,omitempty"`
	SSHKey         string            `json:"ssh_key,omitempty"`
	CloudInit      json.RawMessage   `json:"cloud_init,omitempty"`
}

type InstanceCreateResponse struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type InstanceModifyRequest struct {
	CPUCores int `json:"cpuCores,omitempty"`
	RAMGB    int `json:"ramGb,omitempty"`
	DiskGB   int `json:"diskGb,omitempty"`
	GPUs     *struct {
		GPUV0Name string `json:"gpuV0Name"`
		Count     int    `json:"count"`
	} `json:"gpus,omitempty"`
}

type ActionStatusResponse struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id"`
		Attributes struct {
			Status string `json:"status"`
		} `json:"attributes"`
	} `json:"data"`
}

type MessageResponse struct {
	Data struct {
		Type       string `json:"type"`
		Message    string `json:"message,omitempty"`
		Attributes struct {
			Message string `json:"message"`
		} `json:"attributes"`
	} `json:"data"`
	Message string `json:"message,omitempty"`
}
