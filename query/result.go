package query

import "encoding/json"

// DoctorAuth reports the authentication status.
type DoctorAuth struct {
	Configured bool `json:"configured"`
	Succeeded  bool `json:"succeeded"`
}

// DoctorCapabilities reports which query methods are available.
type DoctorCapabilities struct {
	DBQ              bool `json:"db_q"`
	DatascriptQuery  bool `json:"datascript_query"`
}

// DoctorResult is the structured output of `lsq query doctor`.
type DoctorResult struct {
	Backend      string             `json:"backend"`
	Command      string             `json:"command"`
	APIURL       string             `json:"api_url"`
	Reachable    bool               `json:"reachable"`
	Auth         DoctorAuth         `json:"auth"`
	Capabilities DoctorCapabilities `json:"capabilities"`
	Warnings     []string           `json:"warnings"`
	Error        *string            `json:"error"`
}

// AdvancedResult is the structured output of `lsq query advanced`.
type AdvancedResult struct {
	Backend     string           `json:"backend"`
	InputKind   string           `json:"input_kind"`
	QueryMethod string           `json:"query_method"`
	Results     json.RawMessage  `json:"results"`
	Warnings    []string         `json:"warnings"`
	Error       *string          `json:"error"`
}
