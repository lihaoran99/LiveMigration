package host

import (
	"encoding/json"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/client"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/common"
	"strings"
)

const (
	siteMask = "<site_uri>"
	hostUrl    = "<site_uri>/hosts"
)

type Manager interface {
	ListHost() ([]Host, error)
}

func NewManager(client client.FusionComputeClient, siteUri string) Manager {
	return &manager{client: client, siteUri: siteUri}
}

type manager struct {
	client  client.FusionComputeClient
	siteUri string
}

func (m *manager) ListHost() ([]Host, error) {
	var hosts []Host
	api, err := m.client.GetApiClient()
	if err != nil {
		return nil, err
	}
	request := api.R()
	resp, err := request.Get(strings.Replace(hostUrl, siteMask, m.siteUri, -1))
	if err != nil {
		return nil, err
	}
	if resp.IsSuccess() {
		var listHostResponse ListHostResponse
		err := json.Unmarshal(resp.Body(), &listHostResponse)
		if err != nil {
			return nil, err
		}
		hosts = listHostResponse.Hosts
	} else {
		return nil, common.FormatHttpError(resp)
	}
	return hosts, nil
}