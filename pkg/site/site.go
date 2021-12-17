package site

import (
	"encoding/json"
	"fmt"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/client"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/cluster"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/common"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/helper"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/host"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/vm"
)

const (
	siteUrl = "/service/sites"
)

type Interface interface {
	ListSite() ([]Site, error)
	GetSite(siteUri string) (*Site, error)
}

func NewManager(client client.FusionComputeClient) Interface {
	return &manager{client: client}
}

type manager struct {
	client client.FusionComputeClient
}

func (m *manager) GetSite(siteUri string) (*Site, error) {
	var site Site
	api, err := m.client.GetApiClient()
	if err != nil {
		return nil, err
	}
	resp, err := api.R().Get(siteUri)
	if err != nil {
		return nil, err
	}
	if resp.IsSuccess() {
		err := json.Unmarshal(resp.Body(), &site)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, common.FormatHttpError(resp)
	}
	return &site, nil
}

func (m *manager) ListSite() ([]Site, error) {
	var sites []Site
	api, err := m.client.GetApiClient()
	if err != nil {
		return nil, err
	}
	resp, err := api.R().Get(siteUrl)
	if err != nil {
		return nil, err
	}
	if resp.IsSuccess() {
		var listSiteResponse ListSiteResponse
		err := json.Unmarshal(resp.Body(), &listSiteResponse)
		if err != nil {
			return nil, err
		}
		sites = listSiteResponse.Sites
	} else {
		return nil, common.FormatHttpError(resp)
	}
	return sites, nil
}

func MetaCheckSite(computeClient client.FusionComputeClient, print bool) (Site, []vm.Vm, []host.Host) {
	// 查询站点
	sm := NewManager(computeClient)
	ss, serr := sm.ListSite()
	helper.CheckError(serr)
	s := ss[0]
	if print {
		fmt.Println(s.Uri + ":" + s.Name)
	}
	// 查询所有集群
	cm := cluster.NewManager(computeClient, s.Uri)
	cs, cerr := cm.ListCluster()
	helper.CheckError(cerr)
	if print {
		for _, c := range cs {
			fmt.Println("    " + c.Uri + ":" + c.Name)
		}
	}
	// 查询所有VM
	vmgr := vm.NewManager(computeClient, s.Uri)
	vms, verr := vmgr.ListVm(false)
	helper.CheckError(verr)
	if print {
		for _, v := range vms {
			fmt.Println("        " + v.Uri + ":" + v.Name)
		}
	}
	// 查询所有host
	hostMgr := host.NewManager(computeClient, s.Uri)
	hosts, herr := hostMgr.ListHost()
	helper.CheckError(herr)
	if print {
		for _, h := range hosts {
			fmt.Println("        " + h.Uri + ":" + h.Name)
		}
	}
	if print {
		fmt.Println()
	}
	return s, vms, hosts
}