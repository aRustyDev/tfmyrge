package tfmerge

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

type StateOutput tfjson.StateOutput
type StateModule tfjson.StateModule
type StateResource tfjson.StateResource
type CheckStatus string
type CheckKind string

const (
	CheckStatusPass    CheckStatus = "pass"
	CheckStatusFail    CheckStatus = "fail"
	CheckStatusError   CheckStatus = "error"
	CheckStatusUnknown CheckStatus = "unknown"
)

const (
	CheckKindResource    CheckKind = "resource"
	CheckKindOutputValue CheckKind = "output_value"
	CheckKindCheckBlock  CheckKind = "check"
)

type difference struct {
	expect interface{}
	actual interface{}
}

type differences struct {
	all []difference
}

type State struct {
	Version          int                    `json:"version,omitempty"`
	TerraformVersion string                 `json:"terraform_version,omitempty"`
	Serial           int                    `json:"serial,omitempty"`
	Lineage          string                 `json:"lineage,omitempty"`
	Resources        []Resource             `json:"resources,omitempty"`
	Checks           json.RawMessage        `json:"check_results"`
	Outputs          map[string]interface{} `json:"outputs"`
}

type Resource struct {
	SchemaVersion   uint64          `json:"schema_version,omitempty"`
	Module          string          `json:"module,omitempty"`
	Mode            string          `json:"mode"`
	Type            string          `json:"type"`
	Name            string          `json:"name"`
	Provider        string          `json:"provider"`
	Private         string          `json:"private,omitempty"`
	DependsOn       []string        `json:"dependencies,omitempty"`
	SensitiveValues json.RawMessage `json:"sensitive_attributes,omitempty"`
	Instances       []interface{}   `json:"instances"`
}

type StateValues struct {
	Outputs    tfjson.StateOutput
	RootModule StateModule
}

type CheckResultStatic struct {
	Address   CheckStaticAddress   `json:"address"`
	Status    CheckStatus          `json:"status"`
	Instances []CheckResultDynamic `json:"instances,omitempty"`
}

type CheckStaticAddress struct {
	ToDisplay string    `json:"to_display"`
	Kind      CheckKind `json:"kind"`
	Module    string    `json:"module,omitempty"`
	Mode      string    `json:"mode,omitempty"`
	Type      string    `json:"type,omitempty"`
	Name      string    `json:"name,omitempty"`
}

type CheckResultDynamic struct {
	Address  CheckDynamicAddress  `json:"address"`
	Status   CheckStatus          `json:"status"`
	Problems []CheckResultProblem `json:"problems,omitempty"`
}

type CheckDynamicAddress struct {
	ToDisplay   string      `json:"to_display"`
	Module      string      `json:"module,omitempty"`
	InstanceKey interface{} `json:"instance_key,omitempty"`
}

type CheckResultProblem struct {
	Message string `json:"message"`
}

type session struct { // This just makes it easier to pass ctx & tf to sub functions
	ctx *context.Context
	tf  *tfexec.Terraform
}

type ledger struct { // This struct is used to track what resources are already in the state
	Resource map[string]*tfjson.StateResource
	Children map[string]*tfjson.StateModule
	Roots    map[string]*tfjson.StateModule
}

// ---------------|CONSTRUCTOR FUNC|---------------
// Constructor: pulls version info from first statefile and initializes finalState
func (state *State) init(session session, path string) {
	state_absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Println("error in <State>.init()")
	}
	stateFile, err := session.tf.ShowStateFile(*session.ctx, state_absPath)
	if err != nil {
		fmt.Println("error in <State>.init()")
	}
	state.Version, _ = strconv.Atoi(stateFile.FormatVersion)
	state.TerraformVersion = stateFile.TerraformVersion
}

func (ledger *ledger) init() {
	ledger.Children = make(map[string]*tfjson.StateModule)
	ledger.Resource = make(map[string]*tfjson.StateResource)
	ledger.Roots = make(map[string]*tfjson.StateModule)
}
