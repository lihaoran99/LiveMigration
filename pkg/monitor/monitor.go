package monitor

import (
	"encoding/json"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/client"
	"github.com/KubeOperator/FusionComputeGolangSDK/pkg/common"
	"path"
	"strings"
)

const (
	siteMask = "<site_uri>"
	monitorUrl    = "<site_uri>/monitors"
)

type Manager interface {
	GetObjectMetricRealtimeData(urn string, metricId []string) (* RealtimeData, error)
}

func NewManager(client client.FusionComputeClient, siteUri string) Manager {
	return &manager{client: client, siteUri: siteUri}
}

type manager struct {
	client  client.FusionComputeClient
	siteUri string
}

func (m *manager)GetObjectMetricRealtimeData(urn string, metricId []string) (* RealtimeData, error) {
	var realtimeData RealtimeData
	api, err := m.client.GetApiClient()
	if err != nil {
		return nil, err
	}
	resp, err := api.R().SetBody([]map[string]interface{}{{"urn": urn, "metricId": metricId}}).
		Post(path.Join(strings.Replace(monitorUrl, siteMask, m.siteUri, -1), "/objectmetric-realtimedata"))
	if err != nil {
		return nil, err
	}
	if resp.IsSuccess() {
		err := json.Unmarshal(resp.Body(), &realtimeData)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, common.FormatHttpError(resp)
	}
	return &realtimeData, nil
}