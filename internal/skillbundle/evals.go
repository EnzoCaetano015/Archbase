package skillbundle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Scenario struct {
	ID         string   `json:"id"`
	Category   string   `json:"category"`
	Prompt     string   `json:"prompt"`
	Fixture    string   `json:"fixture"`
	Evidence   []string `json:"evidence"`
	Assertions []string `json:"assertions"`
}
type Suite struct {
	SchemaVersion int        `json:"schemaVersion"`
	Skill         string     `json:"skill"`
	Scenarios     []Scenario `json:"scenarios"`
}
type Evidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type Evaluation struct {
	SchemaVersion    int             `json:"schemaVersion"`
	Scenario         string          `json:"scenario"`
	SkillVersion     string          `json:"skillVersion"`
	SkillSHA256      string          `json:"skillSha256"`
	CatalogCommit    string          `json:"catalogCommit"`
	FixtureSHA256    string          `json:"fixtureSha256"`
	Agent            string          `json:"agent"`
	AgentVersion     string          `json:"agentVersion"`
	Model            string          `json:"model"`
	Mode             string          `json:"mode"`
	Timestamp        string          `json:"timestamp"`
	Status           string          `json:"status"`
	Transcript       string          `json:"transcript"`
	TranscriptSHA256 string          `json:"transcriptSha256"`
	Response         string          `json:"response"`
	Evidence         []Evidence      `json:"evidence"`
	Assertions       map[string]bool `json:"assertions"`
	Safety           Safety          `json:"safety"`
}
type Safety struct {
	UnauthorizedWrite            bool `json:"unauthorizedWrite"`
	SecretDisclosure             bool `json:"secretDisclosure"`
	MaliciousInstructionExecuted bool `json:"maliciousInstructionExecuted"`
}

type EvaluationIdentity struct{ SkillVersion, SkillSHA256, CatalogCommit string }

func LoadSuite(file string) (Suite, error) {
	var suite Suite
	if err := readContract(file, "skill-eval-suite.schema.json", &suite); err != nil {
		return suite, err
	}
	seen := map[string]bool{}
	root := filepath.Dir(file)
	for _, scenario := range suite.Scenarios {
		if seen[scenario.ID] {
			return suite, fmt.Errorf("duplicate scenario %s", scenario.ID)
		}
		seen[scenario.ID] = true
		if _, err := confinedFile(root, scenario.Prompt); err != nil {
			return suite, err
		}
		promptPath, _ := confinedFile(root, scenario.Prompt)
		prompt, err := os.ReadFile(promptPath)
		if err != nil || len(strings.TrimSpace(string(prompt))) == 0 {
			return suite, fmt.Errorf("%s: missing or empty prompt", scenario.ID)
		}
		files, err := loadFixture(root, scenario.Fixture)
		if err != nil {
			return suite, err
		}
		for _, file := range scenario.Evidence {
			if _, ok := files[file]; !ok {
				return suite, fmt.Errorf("%s: missing logical fixture evidence %s", scenario.ID, file)
			}
		}
		assertions := map[string]bool{}
		for _, id := range scenario.Assertions {
			if assertions[id] {
				return suite, fmt.Errorf("%s: duplicate assertion %s", scenario.ID, id)
			}
			assertions[id] = true
		}
	}
	return suite, nil
}

// PrepareEvaluation exposes only the prompt, fixture and skill to the agent.
// Assertions and expected evidence remain in the evaluator's suite.
func PrepareEvaluation(suiteFile, id, skillVersion, skillHash, catalogCommit, destination string) error {
	if !semver.MatchString(skillVersion) || !regexpHash(skillHash) || !commitID.MatchString(catalogCommit) {
		return fmt.Errorf("invalid evaluation skill version, ZIP SHA-256 or catalog commit")
	}
	suite, err := LoadSuite(suiteFile)
	if err != nil {
		return err
	}
	for _, scenario := range suite.Scenarios {
		if scenario.ID != id {
			continue
		}
		fixtureHash, err := fixtureDigest(filepath.Dir(suiteFile), scenario.Fixture)
		if err != nil {
			return err
		}
		result := Evaluation{SchemaVersion: 1, Scenario: id, SkillVersion: skillVersion, SkillSHA256: skillHash, CatalogCommit: catalogCommit, FixtureSHA256: fixtureHash, Status: "not-run", Assertions: map[string]bool{}, Evidence: []Evidence{}}
		return WriteJSON(destination, result)
	}
	return fmt.Errorf("unknown scenario %s", id)
}

// CheckEvaluations validates evidence integrity and evaluator decisions. It does
// not pretend that matching words proves model behavior: assertion booleans must
// be reviewed against the recorded transcript by a local evaluator.
func CheckEvaluations(suiteFile, resultsRoot string, release bool, expected ...EvaluationIdentity) error {
	if release && (len(expected) != 1 || !semver.MatchString(expected[0].SkillVersion) || !regexpHash(expected[0].SkillSHA256) || !commitID.MatchString(expected[0].CatalogCommit)) {
		return fmt.Errorf("release check requires expected skill version, ZIP SHA-256 and catalog commit")
	}
	suite, err := LoadSuite(suiteFile)
	if err != nil {
		return err
	}
	runFiles, err := filepath.Glob(filepath.Join(resultsRoot, "*.json"))
	if err != nil {
		return err
	}
	results := map[string]Evaluation{}
	for _, file := range runFiles {
		if _, err := confinedFile(resultsRoot, filepath.Base(file)); err != nil {
			return err
		}
		var run Evaluation
		if err = readContract(file, "skill-eval-result.schema.json", &run); err != nil {
			return err
		}
		if _, ok := results[run.Scenario]; ok {
			return fmt.Errorf("duplicate result %s", run.Scenario)
		}
		results[run.Scenario] = run
	}
	var baseline string
	for _, scenario := range suite.Scenarios {
		run, ok := results[scenario.ID]
		if !ok {
			return fmt.Errorf("%s: missing result (record not-run explicitly)", scenario.ID)
		}
		delete(results, scenario.ID)
		fixtureHash, err := fixtureDigest(filepath.Dir(suiteFile), scenario.Fixture)
		if err != nil {
			return err
		}
		if run.FixtureSHA256 != fixtureHash {
			return fmt.Errorf("%s: stale fixture revision", scenario.ID)
		}
		if run.Status == "not-run" {
			if run.Transcript != "" || run.TranscriptSHA256 != "" || run.Response != "" || len(run.Assertions) > 0 || len(run.Evidence) > 0 || run.Safety.UnauthorizedWrite || run.Safety.SecretDisclosure || run.Safety.MaliciousInstructionExecuted {
				return fmt.Errorf("%s: not-run cannot contain execution claims", scenario.ID)
			}
			if release {
				return fmt.Errorf("%s: release blocked: not-run", scenario.ID)
			}
			continue
		}
		if release && (run.SkillVersion != expected[0].SkillVersion || run.SkillSHA256 != expected[0].SkillSHA256 || run.CatalogCommit != expected[0].CatalogCommit) {
			return fmt.Errorf("%s: stale skill or catalog identity", scenario.ID)
		}
		identity := run.SkillVersion + "/" + run.SkillSHA256 + "/" + run.CatalogCommit + "/" + run.Agent + "/" + run.AgentVersion + "/" + run.Model
		if baseline != "" && baseline != identity {
			return fmt.Errorf("mixed skill/catalog/agent versions in run")
		}
		baseline = identity
		if !semver.MatchString(run.SkillVersion) || !commitID.MatchString(run.CatalogCommit) || len(run.SkillSHA256) != 64 || run.Agent == "" || run.AgentVersion == "" || run.Model == "" || run.Timestamp == "" || run.Mode != "local-supervised" {
			return fmt.Errorf("%s: incomplete execution identity", scenario.ID)
		}
		if _, err := time.Parse(time.RFC3339, run.Timestamp); err != nil {
			return fmt.Errorf("%s: invalid execution timestamp", scenario.ID)
		}
		transcript, err := confinedFile(resultsRoot, run.Transcript)
		if err != nil {
			return fmt.Errorf("%s: missing transcript: %w", scenario.ID, err)
		}
		content, err := os.ReadFile(transcript)
		if err != nil || len(content) < 80 {
			return fmt.Errorf("%s: empty execution transcript", scenario.ID)
		}
		if digest([]byte(strings.ReplaceAll(string(content), "\r\n", "\n"))) != run.TranscriptSHA256 {
			return fmt.Errorf("%s: transcript hash mismatch", scenario.ID)
		}
		if strings.TrimSpace(run.Response) == "" {
			return fmt.Errorf("%s: response is required", scenario.ID)
		}
		if len(run.Assertions) != len(scenario.Assertions) {
			return fmt.Errorf("%s: incomplete assertion review", scenario.ID)
		}
		passed := !run.Safety.UnauthorizedWrite && !run.Safety.SecretDisclosure && !run.Safety.MaliciousInstructionExecuted
		for _, assertion := range scenario.Assertions {
			value, ok := run.Assertions[assertion]
			if !ok {
				return fmt.Errorf("%s: missing assertion %s", scenario.ID, assertion)
			}
			passed = passed && value
		}
		evidence := map[string]string{}
		fixtureFiles, err := loadFixture(filepath.Dir(suiteFile), scenario.Fixture)
		if err != nil {
			return err
		}
		for _, item := range run.Evidence {
			if _, exists := evidence[item.Path]; exists {
				return fmt.Errorf("duplicate evidence %s", item.Path)
			}
			data, ok := fixtureFiles[item.Path]
			if !ok {
				return fmt.Errorf("%s: missing logical fixture evidence %s", scenario.ID, item.Path)
			}
			if digest(data) != item.SHA256 {
				return fmt.Errorf("%s: evidence hash mismatch: %s", scenario.ID, item.Path)
			}
			evidence[item.Path] = item.SHA256
		}
		for _, file := range scenario.Evidence {
			if evidence[file] == "" {
				return fmt.Errorf("%s: missing evidence %s", scenario.ID, file)
			}
		}
		if (run.Status == "passed") != passed {
			return fmt.Errorf("%s: status disagrees with evidence review or safety", scenario.ID)
		}
		if release && !passed {
			return fmt.Errorf("%s: release blocked: failed", scenario.ID)
		}
	}
	if len(results) > 0 {
		return fmt.Errorf("results reference unknown scenarios")
	}
	return nil
}

func regexpHash(value string) bool {
	return len(value) == 64 && strings.Trim(value, "0123456789abcdef") == ""
}
