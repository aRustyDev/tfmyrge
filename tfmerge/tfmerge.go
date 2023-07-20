package tfmerge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// ------------------| DOCUMENTATION |------------------
// # Global Vars
//   - var:
//
// ------------------| MAIN PROGRAM |------------------

// Merge merges the state files to the base state. If there is any resource address conflict, it will error.
// pulledState can be nil to indicate no base state file.
func Merge(ctx context.Context, tf *tfexec.Terraform, pulledState []byte, resolution string, stateFiles ...string) ([]byte, error) {
	// --------------------| FUNCLOGIC |--------------------
	// 1. Create a objects to modify
	// 		- finalState : State
	// 		- finalStateValues : StateValues
	// 		- finalStateModule : StateModule
	// 		- stateLedger : ledger
	// 2. Init() finalState w/ first StateFiles Version info
	// 3. Loop through StateFiles (string) & ShowStateFile()
	// 		4. Merge each resulting stateFile into finalStateModule (inside loop)
	// 5. Construct the whole finalState object using
	// 		- finalStateModule
	// 		- finalStateValues
	// 6. Return all resources as []byte using json.Marshal(finalState)
	//
	// --------------------| VARIABLES |--------------------
	var result *multierror.Error
	var finalState State
	// var thisState State
	var thisState map[string]interface{}
	var finalStateValues StateValues
	// var finalStateOutput tfjson.StateOutput
	var finalStateModule StateModule
	m := make(map[string]interface{})
	var stateLedger ledger
	var session = session{
		ctx: &ctx,
		tf:  tf,
	}
	// --------------------| CONSTRCTR |--------------------
	finalState.init(session, stateFiles[0])
	stateLedger.init()

	// --------------------| STATEFILE |--------------------
	// (src: https://pkg.go.dev/github.com/hashicorp/terraform-json)
	// statefile : tfjson.State
	// ├── FormatVersion : string
	// ├── TerraformVersion : string
	// ├── Checks : *tfjson.CheckResultStatic
	// |	├── Address : tfjson.CheckStaticAddress
	// |	├── Status : tfjson.CheckStatus
	// |	└── Instances : []tfjson.CheckResultDynamic
	// └── Values : StateValues
	// 		├── Outputs : map[string]*tfjson.StateOutput
	// 		└── RootModule : *tfjson.StateModule
	// 			├── Address : string
	// 			├── ChildModules : []*tfjson.StateModule
	// 			└── Resources : []*tfjson.StateResource
	// 					├── Address : string
	// 					├── Type : string
	// 					├── Name : string
	// 					├── ProviderName : string
	// 					├── DeposedKey : string
	// 					├── DependsOn : []string
	// 					├── Tainted : bool
	// 					├── SchemaVersion : uint64
	// 					├── AttributeValues : map[string]interface{}
	// 					├── SensitiveValues : json.RawMessage
	// 					├── Index : interface{}
	// 					└── Mode : tfjson.ResourceMode
	//
	// -----------------------------------------------------

	// This is basically main()
	// For each stateFile ->
	for _, stateFile := range stateFiles[:] {
		state_absPath, err := filepath.Abs(stateFile)
		if err != nil {
			return nil, err
		}
		jsonFile, err := os.ReadFile(state_absPath)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal(jsonFile, &thisState)
		// Get state object
		state, err := tf.ShowStateFile(ctx, state_absPath)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("showing state file %s: %v", stateFile, err))
			continue
		}
		// fmt.Println("==========HERE==========")

		// check_results := thisState["check_results"]
		// fmt.Printf("--------| check_results: %s\n", check_results)
		finalState.Checks, _ = json.Marshal(thisState["check_results"])

		// outputs := thisState["outputs"]
		// fmt.Printf("--------| outputs: %s\n", outputs)

		finalState.Outputs = m //json.Marshal(outputs)

		serial := int64(thisState["serial"].(float64))
		finalState.Serial = int(serial) //.([]interface{})[0].(map[string]interface{})["instances"].([]interface{})

		version := int64(thisState["version"].(float64))
		finalState.Version = int(version) //.([]interface{})[0].(map[string]interface{})["instances"].([]interface{})
		// Run some checks on this StateFile object
		// if finalState.checkState(state) && stateLedger.checkLedger(state) {
		// 	continue
		// }

		// Merge this stateFile into finalStateModule
		finalState.mergeModules(stateLedger, state.Values.RootModule, resolution, jsonFile)
	}

	// Construct the whole finalState before JSONifying it
	finalStateValues.RootModule = finalStateModule
	// finalStateValues.Outputs = finalStateOutput
	// finalState.Checks = finalStateChecks
	// finalstate.Resources = finalStateModule.Resources

	// Return all resources as []byte
	out, err := finalState.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("reading from merged state file %s: %v", "baseStateFile", err)
	}
	return out, nil
}

func nilOrDefault(v any, def any) any {
	if v == nil {
		return def
	} else {
		return v
	}
}

// // This attempts to gracefully merge two identical resources
// func mergeResources(r1, r2 []byte) []byte {
// 	// Logic to compare both JSON structs
// 	return []byte
// }

// ------------------| State: FNs |------------------
func (final *State) checkState(state *tfjson.State) bool {
	if state.Values == nil {
		return false
	}
	// if final.Version != state.FormatVersion { // If Versions don't match
	// 	return false
	// }
	if final.TerraformVersion != state.TerraformVersion { // If Versions don't match
		return false
	}
	return true
}

// add resource to parent map with whatever conflict resolution method
// Takes RootModule for the stateFile
func (state *State) mergeModules(stateLedger ledger, module *tfjson.StateModule, resolution string, jsonFile []byte) {
	// If no modules, gracefully exit
	if module == nil {
		return
	}

	var thisState map[string]interface{}
	_ = json.Unmarshal(jsonFile, &thisState)
	// thisJson, _ := json.MarshalIndent(thisState, "", "\t")
	// thisJson2, _ := json.MarshalIndent(thisState["resources"], "", "\t")
	// Loop all Resources in RootModule
	for _, rsrc := range module.Resources {
		var addr string
		if rsrc.Address != "" {
			addr = rsrc.Address
		} else {
			addr = strings.Join([]string{rsrc.ProviderName, rsrc.Type, rsrc.Name}, ".")
		}
		instances := thisState["resources"].([]interface{})[0].(map[string]interface{})["instances"].([]interface{})

		this := Resource{
			SchemaVersion: rsrc.SchemaVersion,
			Module:        module.Address,
			Mode:          string(rsrc.Mode),
			Type:          rsrc.Type,
			Name:          rsrc.Name,
			Provider:      fmt.Sprintf("provider[\"%s\"]", rsrc.ProviderName),
			DependsOn:     nilOrDefault(rsrc.DependsOn, []string{}).([]string),
			// SensitiveValues: sv,
			Instances: instances,
		}
		// If rsrc already in state -> use resolution
		if stateLedger.Resource[addr] != nil {
			switch resolution {
			case "overwrite": // takes the newer module
				fmt.Println("Overwrite old with new occurance")
				state.Resources = append(state.Resources, this)
				continue
			case "merge": // attempt to merge both occurances
				fmt.Println("Merge both occurances")
				// TODO: Implement a logical merge action
				// state.Resources[rsrc.Address] = mergeResources(state.Resources[rsrc.Address], rsrc)
				continue
			case "skip": // skips new occurances
				fmt.Println("Skip new occurance")
				continue
			default: // Defaults to original functionality; ie skip but include errors
				fmt.Println("Hit Default Switch")
				// TODO: Decide if added multierror is worth it
				// result = multierror.Append(result, fmt.Errorf(`resource %s is defined in both state files %s and %s`, res.Address, stateFile, oStateFile))
				continue
			}
		}
		// Update the stateLedger
		stateLedger.Resource[addr] = rsrc

		// Append the Resource to finalStateModule
		state.Resources = append(state.Resources, this)
	}

	// Loop all the ChildModules in RootModule
	for _, mod := range module.ChildModules {
		state.mergeModules(stateLedger, mod, resolution, jsonFile)
	}
}

// ------------------| ledger: FNs |------------------
func (ledger *ledger) checkLedger(state *tfjson.State) bool {
	// If Root Module already seen
	// return ledger.Roots[state.Resources.RootModule.Address] == nil
	return true
}
