package monitor

type RealtimeData struct {
	Items		[]item `json:"items"`
}

type item struct {
	Urn			string   `json:"urn"`
	ObjectName  string	 `json:"objectName"`
	Value 		[]value	 `json:"value"`
}

type value struct {
	Unit 		string   `json:"unit"`
	MetricId 	string   `json:"metricId"`
	MetricValue string   `json:"metricValue"`
}

