package cluster

type Cluster struct {
	Uri  			string `json:"uri"`
	Urn  			string `json:"urn"`
	Name 			string `json:"name"`
	Arch 			string `json:"arch"`
	ParentObjUrn	string `json:"parentObjUrn"`
	Description		string `json:"description"`
	Tag				string `json:"tag"`
	IsMemOvercommit	bool   `json:"isMemOvercommit"`
	IsEnableHa		bool   `json:"isEnableHa"`
	IsEnableDrs		bool   `json:"isEnableDrs"`
	ParentObjName	string `json:"parentObjName"`
}

type ListClusterResponse struct {
	Clusters []Cluster `json:"clusters"`
}