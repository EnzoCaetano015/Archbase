package diagnostics

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/EnzoCaetano015/Archbase/schemas"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRuntimeResultsMatchPublicContract(t *testing.T) {
	raw, err := schemas.FS.ReadFile("doctor-result.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err = json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err = compiler.AddResource("doctor.json", document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("doctor.json")
	if err != nil {
		t.Fatal(err)
	}
	valid := tempDir(t)
	malformed := tempDir(t)
	put(t, malformed, ".archbase/scopes.yaml", []byte("scopes: ["))
	for _, root := range []string{valid, malformed, filepath.Join(tempDir(t), "missing")} {
		result := Run(context.Background(), root, "dev", Options{})
		data, _ := json.Marshal(result)
		var value map[string]any
		json.Unmarshal(data, &value)
		if err = schema.Validate(value); err != nil {
			t.Fatal(err)
		}
		checks := value["checks"].([]any)
		checks[0].(map[string]any)["id"] = checks[1].(map[string]any)["id"]
		if schema.Validate(value) == nil {
			t.Fatal("duplicate/missing check ID accepted")
		}
	}
	// Unknown fields and unsafe paths are not extension points of this contract.
	result := Run(context.Background(), valid, "dev", Options{})
	data, _ := json.Marshal(result)
	var value map[string]any
	json.Unmarshal(data, &value)
	value["timestamp"] = "must-not-be-added"
	if schema.Validate(value) == nil {
		t.Fatal("unknown field accepted")
	}
	delete(value, "timestamp")
	value["checks"].([]any)[0].(map[string]any)["path"] = "../secret"
	if schema.Validate(value) == nil {
		t.Fatal("unsafe path accepted")
	}
}
