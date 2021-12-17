package host

type Host struct {
    Uri                     string `json:"uri,omitempty"`
    Urn                     string `json:"urn,omitempty"`
    Name                    string `json:"name,omitempty"`
    Description             string `json:"description,omitempty"`
    IP                      string `json:"ip,omitempty"`
    BcmIP                   string `json:"bcmIp,omitempty"`
    BcmUserName             string `json:"bcmUserName,omitempty"`
    ClusterUrn              string `json:"clusterUrn,omitempty"`
    ClusterName             string `json:"clusterName,omitempty"`
    Status                  string `json:"status,omitempty"`
    IsMaintaining           bool   `json:"isMaintaining,omitempty"`
    MultiPathMode           string `json:"multiPathMode,omitempty"`
    HostMultiPathMode       string `json:"hostMultiPathMode,omitempty"`
    MemQuantityMB           int    `json:"memQuantityMB,omitempty"`
    CpuQuantity             int    `json:"cpuQuantity,omitempty"`
    CpuMHz                  int    `json:"cpuMHz,omitempty"`
    NicQuantity             int    `json:"nicQuantity,omitempty"`
    AttachedISOVMs          []string `json:"attachedISOVMs,omitempty"`
    ComputeResourceStatics  string `json:"computeResourceStatics,omitempty"`
    NtpIP1                  string `json:"ntpIp1,omitempty"`
    NtpIP2                  string `json:"ntpIp2,omitempty"`
    NtpIP3                  string `json:"ntpIp3,omitempty"`
    NtpCycle                int    `json:"ntpCycle,omitempty"`
    PhysicalCpuQuantity     int    `json:"physicalCpuQuantity,omitempty"`
}

type ListHostResponse struct {
    Total int  `json:"total,omitempty"`
    Hosts   []Host `json:"hosts,omitempty"`
}