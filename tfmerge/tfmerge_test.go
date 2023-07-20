package tfmerge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/stretchr/testify/require"
)

func initTest(ctx context.Context, t *testing.T) *tfexec.Terraform {
	// Discard log output
	log.SetOutput(io.Discard)

	// Init terraform with null provider
	dir := t.TempDir()
	i := install.NewInstaller()
	tfpath, err := i.Ensure(ctx, []src.Source{
		&fs.Version{
			Product:     product.Terraform,
			Constraints: version.MustConstraints(version.NewConstraint(">=1.1.0")),
		},
	})
	if err != nil {
		t.Fatalf("finding a terraform executable: %v", err)
	}
	tf, err := tfexec.NewTerraform(dir, tfpath)
	if err != nil {
		t.Fatalf("error running NewTerraform: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "terraform.tf"), []byte(`terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
    }
  }
}
`), 0644); err != nil {
		t.Fatal(err)
	}

	if err := tf.Init(ctx); err != nil {
		t.Fatal(err)
	}

	return tf
}

func testFixture(t *testing.T, name string) (stateFiles []string, expectState []byte) {
	dir := filepath.Join("./testdata", name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir entries: %v", err)
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.Name() == "expect" {
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading file %s: %v", path, err)
			}
			expectState = b
			continue
		}
		stateFiles = append(stateFiles, path)
	}
	return
}

// Initialize the State for testing (basically a constructor)
func (state *State) init_test(t *testing.T, input []byte, mergedCount int, hasBaseState bool) {
	if err := json.Unmarshal(input, &state); err != nil {
		t.Fatalf("unmarshal expect state\n%s\n: %v", string(input), err)
	}

	if !hasBaseState {
		state.Lineage = "00000000-0000-0000-0000-000000000000"
	}

	// TODO: explain why/what this is
	if hasBaseState {
		mergedCount += 1
	}
	state.Serial = mergedCount

	// The terraform version used to create the testdata might be different than the one running this test.
	state.TerraformVersion = ""
}

// Compare both states, return flat map of each difference
func compareStates(t *testing.T, input, expected *State) ([]difference, []error) {
	var diffs differences
	var errs []error
	actualPtr := reflect.ValueOf(input)
	actual := reflect.Indirect(actualPtr)
	expectPtr := reflect.ValueOf(expected)
	expect := reflect.Indirect(expectPtr)

	// different field sizes
	if actual.NumField() != expect.NumField() {
		errmsg := fmt.Sprintf("---| SizeMismatch |---\n--| Actual: %v\n--| Expect: %v\n", actual.NumField(), expect.NumField())
		errs = append(errs, errors.New(errmsg))
		return nil, errs
	}

	// walk each State
	for i := 0; i < actual.NumField(); i++ {
		switch actual.Type().Field(i).Name {
		case "Resources":
			errs = append(errs, diffs.compareResources(t, input.Resources, expected.Resources)...)
		case "Checks":
			require.JSONEq(t, string(expected.Checks), string(input.Checks))
		case "Outputs":
			if !reflect.DeepEqual(actual, expected) {
				errmsg := fmt.Sprintf("---| DeepEqualFailure |---\n--| Actual: %v\n--| Expect: %v\n", actual, expected)
				errs = append(errs, errors.New(errmsg))
			}
		default:
			errs = append(errs, diffs.compareField(t, actual, expect, i)...)
		}
	}

	return diffs.all, errs
}

// Updates the differences obj; returns []err
func (diffs *differences) compareField(t *testing.T, actual, expect reflect.Value, i int) []error {
	var errs []error
	if actual.Field(i).Type() != expect.Field(i).Type() { // The Types mismatched
		errmsg := fmt.Sprintf("---| TypeMismatch |---\n--| Actual: %v\n--| Expect: %v\n", actual.Field(i).Type(), expect.Field(i).Type())
		errs = append(errs, errors.New(errmsg))
	}
	if actual.Field(i).Interface() != expect.Field(i).Interface() { // The Values mismatched
		diffs.all = append(diffs.all, difference{
			expect: expect.Field(i).Interface(),
			actual: actual.Field(i).Interface(),
		})
		errmsg := fmt.Sprintf("---| ValueMismatch |---\n--| Actual: %v\n--| Expect: %v\n", actual.Field(i).Interface(), expect.Field(i).Interface())
		errs = append(errs, errors.New(errmsg))
	}
	return errs
}

func (diffs *differences) compareResources(t *testing.T, input, expected []Resource) []error {
	var errs []error
	// For every resource
	for i := range input {
		actualPtr := reflect.ValueOf(input[i])
		actual := reflect.Indirect(actualPtr)
		expectPtr := reflect.ValueOf(expected[i])
		expect := reflect.Indirect(expectPtr)

		// For every property in the resource
		for k := 0; k < expect.NumField(); k++ {
			switch actual.Type().Field(k).Name {
			default:
				errs = append(errs, diffs.compareField(t, actual, expect, i)...)
			}
		}
	}
	return errs
}

// This is Main()
func TestMerge(t *testing.T) {
	var actualState, expectState State

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()                                                   // Set context
			tf := initTest(ctx, t)                                                        // terraform init
			stateFiles, expect := testFixture(t, tt.dir)                                  // Grabs the StateFiles & the Expected State
			actual, err := Merge(ctx, tf, []byte(tt.baseState), "default", stateFiles...) // Run Merge()
			if tt.hasError {
				require.Error(t, err)
				return
			}
			// Update each state struct
			actualState.init_test(t, actual, len(stateFiles), tt.baseState != "")
			expectState.init_test(t, expect, len(stateFiles), tt.baseState != "")

			// Compare the States
			if diffs, errs := compareStates(t, &actualState, &expectState); len(diffs) > 0 && len(errs) > 0 {
				for _, diff := range diffs {
					fmt.Printf("============================================\n")
					fmt.Printf("--| Actual |--\n%v\n", diff.actual)
					fmt.Printf("--| Expect |--\n%v\n", diff.expect)
				}
			} else {
				for _, err := range errs {
					fmt.Printf("%v", err)
				}
			}

			// require.NoError(t, err)
			// assertStateEqual(t, actual, expect, len(stateFiles), tt.baseState != "")

		})
	}
}

// For each case
// Run each test
//
