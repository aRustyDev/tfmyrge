package tfmerge

import (
	"encoding/json"
	"fmt"
)

// ---------------|JSON MARSHALLING|---------------

func (state *State) MarshalJSON() ([]byte, error) {
	values, _ := json.Marshal(state.Resources)
	// checks, _ := state.Checks.MarshalJSON()
	// outputs, _ := state.Outputs

	return json.Marshal(&struct {
		Checks           json.RawMessage        `json:"check_results"`
		Version          int                    `json:"version"`
		TerraformVersion string                 `json:"terraform_version,omitempty"`
		Serial           int                    `json:"serial"`
		Lineage          string                 `json:"lineage,omitempty"` //364d8449-e325-c78f-132a-c1c5791fec40
		Outputs          map[string]interface{} `json:"outputs"`
		Resources        json.RawMessage        `json:"resources,omitempty"`
	}{
		Checks:           state.Checks,
		Version:          state.Version,
		Serial:           state.Serial,
		TerraformVersion: state.TerraformVersion,
		Resources:        values,
		Outputs:          state.Outputs,
	})
}

func (sv *StateValues) MarshalJSON() ([]byte, error) {
	rootmod, err := json.Marshal(sv.RootModule)
	if err != nil {
		return nil, fmt.Errorf("reading from merged state file %s: %v", "---|StateValues.RootModule.MarshalJSON()|---", err)
	}
	return json.Marshal(&struct {
		Outputs    json.RawMessage `json:"outputs,omitempty"`
		RootModule json.RawMessage `json:"root_module,omitempty"`
	}{
		// Outputs:    output,
		RootModule: rootmod,
	})
}

func (sm *StateModule) MarshalJSON() ([]byte, error) {
	instances, err := json.Marshal(sm.Resources)
	if err != nil {
		return nil, fmt.Errorf("reading from merged state file %s: %v", "---|StateModule.Resources.MarshalJSON()|---", err)
	}
	return json.Marshal(&struct {
		Module    string          `json:"module,omitempty"`
		Mode      string          `json:"mode,omitempty"`
		Type      string          `json:"type,omitempty"`
		Name      string          `json:"name,omitempty"`
		Provider  string          `json:"provider,omitempty"`
		Instances json.RawMessage `json:"instances,omitempty"`
	}{
		Module:    "",
		Mode:      "",
		Type:      "",
		Name:      "",
		Provider:  "",
		Instances: instances,
	})
}

func (sr *StateResource) MarshalJSON() ([]byte, error) {
	attr, err := json.Marshal(sr.AttributeValues)
	if err != nil {
		return nil, fmt.Errorf("reading from merged state file %s: %v", "---|StateResource.AttributeValues.MarshalJSON()|---", err)
	}
	secrets, err := sr.SensitiveValues.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("reading from merged state file %s: %v", "---|StateResource.SensitiveValues.MarshalJSON()|---", err)
	}
	return json.Marshal(&struct {
		DependsOn       []string        `json:"dependencies,omitempty"`
		Private         string          `json:"private,omitempty"`
		SchemaVersion   uint64          `json:"schema_version,omitempty"`
		AttributeValues json.RawMessage `json:"attributes,omitempty"`
		SensitiveValues json.RawMessage `json:"sensitive_attributes,omitempty"`
	}{
		DependsOn:       []string{""},
		Private:         "",
		SchemaVersion:   0,
		AttributeValues: attr,
		SensitiveValues: secrets,
	})
}
