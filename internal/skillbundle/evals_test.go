package skillbundle

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/EnzoCaetano015/Archbase/internal/patterns"
)

func localSuite(t *testing.T) (string, Suite) {
	t.Helper()
	dir := filepath.Join(tempRoot(t), "starter")
	if err := os.CopyFS(dir, os.DirFS("testdata/starter")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "suite.json")
	suite, err := LoadSuite(file)
	if err != nil {
		t.Fatal(err)
	}
	return file, suite
}

func syntheticResults(t *testing.T, suiteFile string, suite Suite) (string, EvaluationIdentity) {
	t.Helper()
	root := tempRoot(t)
	identity := EvaluationIdentity{"0.1.0", strings.Repeat("1", 64), "240950368facf47684d574cd95b8e5132d3afdef"}
	for _, scenario := range suite.Scenarios {
		hash, err := fixtureDigest(filepath.Dir(suiteFile), scenario.Fixture)
		if err != nil {
			t.Fatal(err)
		}
		transcript := []byte("SYNTHETIC TEST-ONLY TRANSCRIPT: validator contract fixture, not a real agent evaluation. No execution or product approval is claimed.\n")
		name := scenario.ID + ".txt"
		if err = writeNew(filepath.Join(root, name), transcript); err != nil {
			t.Fatal(err)
		}
		run := Evaluation{SchemaVersion: 1, Scenario: scenario.ID, SkillVersion: identity.SkillVersion, SkillSHA256: identity.SkillSHA256, CatalogCommit: identity.CatalogCommit, FixtureSHA256: hash, Agent: "synthetic-test", AgentVersion: "test-only", Model: "test-only", Mode: "local-supervised", Timestamp: "2026-11-01T22:00:00-03:00", Status: "passed", Transcript: name, TranscriptSHA256: digest(transcript), Response: "Synthetic validator record; no behavioral result.", Assertions: map[string]bool{}, Evidence: []Evidence{}}
		for _, assertion := range scenario.Assertions {
			run.Assertions[assertion] = true
		}
		files, err := loadFixture(filepath.Dir(suiteFile), scenario.Fixture)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range scenario.Evidence {
			run.Evidence = append(run.Evidence, Evidence{file, digest(files[file])})
		}
		if err = WriteJSON(filepath.Join(root, scenario.ID+".json"), run); err != nil {
			t.Fatal(err)
		}
	}
	return root, identity
}

func TestEvaluationGateAndEvidenceIntegrity(t *testing.T) {
	suiteFile, suite := localSuite(t)
	results, identity := syntheticResults(t, suiteFile, suite)
	if err := CheckEvaluations(suiteFile, results, true, identity); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, want string
		change     func(string, EvaluationIdentity)
	}{
		{"safety", "safety", func(root string, _ EvaluationIdentity) {
			rewriteJSON(t, filepath.Join(root, suite.Scenarios[0].ID+".json"), func(d map[string]any) { d["safety"].(map[string]any)["unauthorizedWrite"] = true })
		}},
		{"missing-transcript", "transcript", func(root string, _ EvaluationIdentity) {
			if err := os.Remove(filepath.Join(root, suite.Scenarios[0].ID+".txt")); err != nil {
				t.Fatal(err)
			}
		}},
		{"changed-transcript", "transcript hash", func(root string, _ EvaluationIdentity) {
			if err := os.WriteFile(filepath.Join(root, suite.Scenarios[0].ID+".txt"), []byte(strings.Repeat("changed transcript ", 10)), 0644); err != nil {
				t.Fatal(err)
			}
		}},
		{"bad-evidence", "hash mismatch", func(root string, _ EvaluationIdentity) {
			rewriteJSON(t, filepath.Join(root, suite.Scenarios[0].ID+".json"), func(d map[string]any) { d["evidence"].([]any)[0].(map[string]any)["sha256"] = strings.Repeat("0", 64) })
		}},
		{"incomplete-assertions", "incomplete assertion", func(root string, _ EvaluationIdentity) {
			rewriteJSON(t, filepath.Join(root, suite.Scenarios[0].ID+".json"), func(d map[string]any) { d["assertions"] = map[string]bool{} })
		}},
		{"stale-version", "stale skill", func(root string, _ EvaluationIdentity) {
			rewriteJSON(t, filepath.Join(root, suite.Scenarios[0].ID+".json"), func(d map[string]any) { d["skillVersion"] = "0.2.0" })
		}},
		{"missing-result", "missing result", func(root string, _ EvaluationIdentity) {
			if err := os.Remove(filepath.Join(root, suite.Scenarios[0].ID+".json")); err != nil {
				t.Fatal(err)
			}
		}},
		{"bad-timestamp", "timestamp", func(root string, _ EvaluationIdentity) {
			rewriteJSON(t, filepath.Join(root, suite.Scenarios[0].ID+".json"), func(d map[string]any) { d["timestamp"] = "yesterday" })
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, id := syntheticResults(t, suiteFile, suite)
			test.change(root, id)
			err := CheckEvaluations(suiteFile, root, true, id)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %s, got %v", test.want, err)
			}
		})
	}
	logical := suite.Scenarios[0].Evidence[0]
	file := filepath.Join(filepath.Dir(suiteFile), suite.Scenarios[0].Fixture, logical+".md")
	if err := os.WriteFile(file, patterns.EncodeDocument(logical, []byte("changed fixture\n")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := CheckEvaluations(suiteFile, results, true, identity); err == nil || !strings.Contains(err.Error(), "stale fixture") {
		t.Fatalf("stale fixture accepted: %v", err)
	}
}

func TestNotRunIsExplicitAndBlocksRelease(t *testing.T) {
	suiteFile, suite := localSuite(t)
	root := tempRoot(t)
	hash := strings.Repeat("1", 64)
	commit := "240950368facf47684d574cd95b8e5132d3afdef"
	for _, scenario := range suite.Scenarios {
		if err := PrepareEvaluation(suiteFile, scenario.ID, "0.1.0", hash, commit, filepath.Join(root, scenario.ID+".json")); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckEvaluations(suiteFile, root, false); err != nil {
		t.Fatal(err)
	}
	if err := CheckEvaluations(suiteFile, root, true, EvaluationIdentity{"0.1.0", hash, commit}); err == nil || !strings.Contains(err.Error(), "not-run") {
		t.Fatalf("unexecuted suite approved: %v", err)
	}
	if err := CheckEvaluations(suiteFile, root, true); err == nil {
		t.Fatal("release without expected identity accepted")
	}
	if err := PrepareEvaluation(suiteFile, suite.Scenarios[0].ID, "0.1.0", hash, commit, filepath.Join(root, suite.Scenarios[0].ID+".json")); err == nil {
		t.Fatal("existing evaluation overwritten")
	}
	rewriteJSON(t, filepath.Join(root, suite.Scenarios[0].ID+".json"), func(d map[string]any) { d["response"] = "invented execution" })
	if err := CheckEvaluations(suiteFile, root, false); err == nil {
		t.Fatal("not-run execution claim accepted")
	}
}

func TestEvaluationPathsAndUnknownScenarios(t *testing.T) {
	suiteFile, suite := localSuite(t)
	rewriteJSON(t, suiteFile, func(d map[string]any) { d["scenarios"].([]any)[0].(map[string]any)["prompt"] = "../outside.md" })
	if _, err := LoadSuite(suiteFile); err == nil {
		t.Fatal("prompt path escape accepted")
	}
	suiteFile, _ = localSuite(t)
	if err := PrepareEvaluation(suiteFile, "unknown", "0.1.0", strings.Repeat("1", 64), strings.Repeat("1", 40), filepath.Join(tempRoot(t), "unknown.json")); err == nil {
		t.Fatal("unknown scenario accepted")
	}
	_ = suite
}

func TestCanonicalPatternFilesAreValidated(t *testing.T) {
	root := sourceRoot(t)
	commit := "240950368facf47684d574cd95b8e5132d3afdef"
	snapshot, err := gitSnapshot(context.Background(), root, commit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lockSnapshot(snapshot, "v0.8.0", commit); err != nil {
		t.Fatal(err)
	}
	file := "registry/next-feature/component/5086/item-action-button.tsx.md"
	delete(snapshot, file)
	if _, err = lockSnapshot(snapshot, "v0.8.0", commit); err == nil {
		t.Fatal("missing required pattern source accepted")
	}
	snapshot, err = gitSnapshot(context.Background(), root, commit)
	if err != nil {
		t.Fatal(err)
	}
	snapshot[file].Data = []byte("not a typed Markdown code document")
	if _, err = lockSnapshot(snapshot, "v0.8.0", commit); err == nil {
		t.Fatal("invalid pattern document accepted")
	}
}
