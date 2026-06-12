package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const upstreamRef = "eddc5c75ed527a8348bfcaa85692e53189600833"

type proofCase struct {
	ID                        string `json:"id"`
	ExpectedNativeContextBool bool   `json:"expectedNativeContextInjection"`
	ObservedItemKind          string `json:"observedItemKind"`
	ObservedRole              string `json:"observedRole,omitempty"`
	ContainsSentinel          bool   `json:"containsSentinel"`
	Pass                      bool   `json:"pass"`
}

type projectionFixture struct {
	Fragments map[string]projectedFragment `json:"fragments"`
}

type projectedFragment struct {
	ID                        string `json:"id"`
	SourceID                  string `json:"sourceID"`
	Target                    string `json:"target"`
	ExpectedChannel           string `json:"expectedChannel"`
	ExpectedNativeContextBool bool   `json:"expectedNativeContextInjection"`
	ProofRequired             bool   `json:"proofRequired"`
	ExpectedItemKind          string `json:"expectedItemKind"`
	ExpectedRole              string `json:"expectedRole,omitempty"`
	Sentinel                  string `json:"sentinel"`
	RenderedBody              string `json:"renderedBody"`
}

type promptRoute struct {
	ID                 string   `json:"id"`
	Class              string   `json:"class"`
	SelectedFragments  []string `json:"selectedFragments"`
	RequiresDerivation bool     `json:"requiresDerivation"`
	Confidence         float64  `json:"confidence"`
	Hints              struct {
		Objective   string   `json:"objective,omitempty"`
		Constraints []string `json:"constraints,omitempty"`
		NextActions []string `json:"nextActions,omitempty"`
	} `json:"hints,omitempty"`
	EmitsFullRegistry bool `json:"emitsFullRegistry"`
	AdditionalContext struct {
		Channel                string `json:"channel"`
		ItemKind               string `json:"itemKind"`
		NativeContextInjection bool   `json:"nativeContextInjection"`
	} `json:"additionalContext"`
}

type derivationCase struct {
	ID          string `json:"id"`
	RouteID     string `json:"routeID"`
	OutputValid bool   `json:"outputValid"`
	Accepted    bool   `json:"accepted"`
	Fallback    string `json:"fallback"`
	Injection   struct {
		Channel                          string `json:"channel"`
		ItemKind                         string `json:"itemKind"`
		NativeContextInjection           bool   `json:"nativeContextInjection"`
		ToolResultNativeContextInjection bool   `json:"toolResultNativeContextInjection"`
	} `json:"injection"`
	Derivation *promptDerivation `json:"derivation,omitempty"`
}

type promptDerivation struct {
	RouteID           string   `json:"routeID"`
	AdvisoryOnly      bool     `json:"advisoryOnly"`
	Source            string   `json:"source"`
	Summary           string   `json:"summary"`
	SelectedFragments []string `json:"selectedFragments"`
	Hints             struct {
		Objective            string   `json:"objective"`
		Constraints          []string `json:"constraints"`
		SuggestedNextActions []string `json:"suggestedNextActions"`
		ForbiddenAssumptions []string `json:"forbiddenAssumptions"`
	} `json:"hints"`
	Validity struct {
		CUEValidated bool `json:"cueValidated"`
	} `json:"validity"`
}

type stage2Proof struct {
	ID   string `json:"id"`
	Pass bool   `json:"pass"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "validate":
		err = validate()
	case "generate":
		err = generate()
	case "patch-upstream":
		err = patchUpstream()
	case "test-upstream":
		err = testUpstream()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "poc: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`usage: poc <command>

commands:
  validate        validate generated fixtures and CUE contracts
  generate        regenerate JSON fixtures from the built-in simulated registry model
  patch-upstream  apply patches/openai-codex/*.patch to upstream/openai-codex
  test-upstream   run the narrow Codex runtime proof test target`)
}

func validate() error {
	for _, path := range []string{
		"generated/context_projection.json",
		"generated/proof_cases.json",
		"generated/hook_prompt_hints.json",
		"generated/prompt_routes.json",
		"generated/prompt_derivation_cases.json",
		"generated/stage2_expected_report.json",
		"testdata/expected_report.json",
	} {
		if err := validateJSON(path); err != nil {
			return err
		}
	}

	if err := validateProofConsistency(); err != nil {
		return err
	}
	if err := validateStage2(); err != nil {
		return err
	}

	if _, err := exec.LookPath("cue"); err == nil {
		cueChecks := [][]string{
			{"vet", "./contract/..."},
			{"vet", "generated/prompt_routes.json", "contract/prompt/route.cue", "-d", "#PromptRoutes"},
			{"vet", "generated/prompt_derivation_cases.json", "contract/prompt/route.cue", "contract/prompt/derivation.cue", "-d", "#PromptDerivationCases"},
			{"vet", "generated/stage2_expected_report.json", "contract/prompt/stage2_report.cue", "-d", "#Stage2ExpectedReport"},
		}
		for _, args := range cueChecks {
			cmd := exec.Command("cue", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("cue %s failed: %w", strings.Join(args, " "), err)
			}
		}
	} else {
		return errors.New("cue not found")
	}

	fmt.Println("validation passed")
	return nil
}

func validateStage2() error {
	var projection projectionFixture
	if err := readJSON("generated/context_projection.json", &projection); err != nil {
		return err
	}
	var routes []promptRoute
	if err := readJSON("generated/prompt_routes.json", &routes); err != nil {
		return err
	}
	var derivations []derivationCase
	if err := readJSON("generated/prompt_derivation_cases.json", &derivations); err != nil {
		return err
	}
	var report struct {
		Proofs []stage2Proof `json:"proofs"`
		Pass   bool          `json:"pass"`
	}
	if err := readJSON("generated/stage2_expected_report.json", &report); err != nil {
		return err
	}

	routeByID := make(map[string]promptRoute, len(routes))
	for _, route := range routes {
		if route.ID == "" || routeByID[route.ID].ID != "" {
			return fmt.Errorf("prompt route id %q is empty or duplicated", route.ID)
		}
		routeByID[route.ID] = route
		if route.EmitsFullRegistry {
			return fmt.Errorf("%s emits the full registry", route.ID)
		}
		if route.AdditionalContext.Channel != "message" ||
			route.AdditionalContext.ItemKind != "message" ||
			!route.AdditionalContext.NativeContextInjection {
			return fmt.Errorf("%s additionalContext is not native message context", route.ID)
		}
		if len(route.SelectedFragments) == 0 {
			return fmt.Errorf("%s selects no fragments", route.ID)
		}
		seen := map[string]bool{}
		for _, id := range route.SelectedFragments {
			if _, ok := projection.Fragments[id]; !ok {
				return fmt.Errorf("%s selects undeclared fragment %q", route.ID, id)
			}
			if seen[id] {
				return fmt.Errorf("%s selects fragment %q more than once", route.ID, id)
			}
			seen[id] = true
		}
		if route.Confidence < 0 || route.Confidence > 1 {
			return fmt.Errorf("%s confidence %v is outside [0,1]", route.ID, route.Confidence)
		}
		encoded, _ := json.Marshal(route.Hints)
		if len(encoded) > 2048 {
			return fmt.Errorf("%s route hints exceed compact output bound", route.ID)
		}
	}

	validDerivation := false
	invalidFallback := false
	for _, c := range derivations {
		route, ok := routeByID[c.RouteID]
		if !ok {
			return fmt.Errorf("%s references unknown route %q", c.ID, c.RouteID)
		}
		if c.Injection.ToolResultNativeContextInjection {
			return fmt.Errorf("%s classifies tool output as native context", c.ID)
		}
		if !route.RequiresDerivation && c.Accepted {
			return fmt.Errorf("%s runs derivation for route %q that does not require it", c.ID, c.RouteID)
		}
		if !c.OutputValid {
			if c.Accepted || c.Fallback != "route-only" || c.Injection.NativeContextInjection {
				return fmt.Errorf("%s does not discard invalid derivation output", c.ID)
			}
			invalidFallback = true
			continue
		}
		if !c.Accepted || c.Fallback != "none" || c.Derivation == nil {
			return fmt.Errorf("%s does not accept valid derivation output", c.ID)
		}
		if !c.Derivation.AdvisoryOnly || !c.Derivation.Validity.CUEValidated {
			return fmt.Errorf("%s derivation is authoritative or not CUE validated", c.ID)
		}
		if c.Injection.Channel != "message" || c.Injection.ItemKind != "message" || !c.Injection.NativeContextInjection {
			return fmt.Errorf("%s vetted hints are not injected as message context", c.ID)
		}
		encoded, _ := json.Marshal(c.Derivation)
		if len(encoded) > 2048 {
			return fmt.Errorf("%s derivation exceeds resolver output bound", c.ID)
		}
		for _, id := range c.Derivation.SelectedFragments {
			if _, ok := projection.Fragments[id]; !ok {
				return fmt.Errorf("%s derives undeclared fragment %q", c.ID, id)
			}
		}
		validDerivation = true
	}
	if !validDerivation || !invalidFallback {
		return errors.New("stage2 derivation fixtures must include accepted valid output and discarded invalid output")
	}

	expectedProofs := []string{
		"prompt_route_declared_fragments",
		"prompt_route_no_registry_delivery",
		"prompt_route_additional_context_is_message",
		"sdk_derivation_optional",
		"sdk_derivation_schema_valid",
		"sdk_derivation_advisory_only",
		"sdk_derivation_compact_hint_only",
		"sdk_derivation_not_tool_context",
		"mcp_registry_resource_rejected_as_context",
	}
	if !report.Pass || len(report.Proofs) != len(expectedProofs) {
		return errors.New("stage2 expected report is incomplete or failing")
	}
	for i, id := range expectedProofs {
		if report.Proofs[i].ID != id || !report.Proofs[i].Pass {
			return fmt.Errorf("stage2 proof %d mismatch: got %#v want %q passing", i, report.Proofs[i], id)
		}
	}
	return nil
}

func validateProofConsistency() error {
	var projection projectionFixture
	if err := readJSON("generated/context_projection.json", &projection); err != nil {
		return err
	}
	proofCases, err := readProofCases("generated/proof_cases.json")
	if err != nil {
		return err
	}
	reportCases, err := readProofCases("testdata/expected_report.json")
	if err != nil {
		return err
	}

	expectedProofCases := proofCasesFromProjection(projection.Fragments, true)
	if err := compareProofCases("generated/proof_cases.json", proofCases, expectedProofCases); err != nil {
		return err
	}
	if err := compareProofCases("testdata/expected_report.json", reportCases, expectedProofCases); err != nil {
		return err
	}
	if err := validateProjectionSemantics(projection.Fragments); err != nil {
		return err
	}
	return nil
}

func validateProjectionSemantics(frags map[string]projectedFragment) error {
	for id, frag := range frags {
		wantChannel := "message"
		wantItemKind := "message"
		if frag.Target == "json_tool_output" || frag.Target == "mcp_tool_output" {
			wantChannel = "tool_output"
			wantItemKind = "function_call_output"
		}
		if frag.ExpectedChannel != wantChannel {
			return fmt.Errorf("%s expectedChannel=%q want %q", id, frag.ExpectedChannel, wantChannel)
		}
		if frag.ExpectedItemKind != wantItemKind {
			return fmt.Errorf("%s expectedItemKind=%q want %q", id, frag.ExpectedItemKind, wantItemKind)
		}
		if frag.ExpectedNativeContextBool != (wantChannel == "message" && wantItemKind == "message") && frag.Target != "turn_start_additional_context" {
			return fmt.Errorf("%s expectedNativeContextInjection=%v inconsistent with channel/itemKind", id, frag.ExpectedNativeContextBool)
		}
	}
	return nil
}

func validateJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("invalid JSON %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid JSON %s: %w", path, err)
	}
	return nil
}

func readProofCases(path string) ([]proofCase, error) {
	var cases []proofCase
	if err := readJSON(path, &cases); err != nil {
		var report struct {
			Cases []proofCase `json:"cases"`
		}
		if err2 := readJSON(path, &report); err2 != nil {
			return nil, err
		}
		return report.Cases, nil
	}
	return cases, nil
}

func compareProofCases(path string, actual, expected []proofCase) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("%s has %d cases, want %d", path, len(actual), len(expected))
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return fmt.Errorf("%s case %d mismatch: got %#v want %#v", path, i, actual[i], expected[i])
		}
	}
	return nil
}

func proofCasesFromProjection(frags map[string]projectedFragment, proofRequired bool) []proofCase {
	type rankedFragment struct {
		key  string
		frag projectedFragment
		rank int
	}
	items := make([]rankedFragment, 0, len(frags))
	for id, frag := range frags {
		if frag.ProofRequired == proofRequired {
			items = append(items, rankedFragment{key: id, frag: frag, rank: targetRank(frag.Target)})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].rank != items[j].rank {
			return items[i].rank < items[j].rank
		}
		return items[i].key < items[j].key
	})
	cases := make([]proofCase, 0, len(items))
	for _, item := range items {
		frag := item.frag
		cases = append(cases, proofCase{
			ID:                        targetCaseID(frag.Target),
			ExpectedNativeContextBool: frag.ExpectedNativeContextBool,
			ObservedItemKind:          frag.ExpectedItemKind,
			ObservedRole:              frag.ExpectedRole,
			ContainsSentinel:          true,
			Pass:                      true,
		})
	}
	return cases
}

func targetRank(target string) int {
	switch target {
	case "internal_model_context":
		return 0
	case "available_skills":
		return 1
	case "hook_prompt_fragment":
		return 2
	case "json_tool_output":
		return 3
	case "mcp_tool_output":
		return 4
	default:
		return 100
	}
}

func targetCaseID(target string) string {
	switch target {
	case "internal_model_context":
		return "internal_model_context"
	case "available_skills":
		return "available_skills"
	case "hook_prompt_fragment":
		return "hook_prompt_fragment"
	case "json_tool_output":
		return "json_tool_output"
	case "mcp_tool_output":
		return "mcp_tool_output"
	case "turn_start_additional_context":
		return "turn_start_additional_context"
	default:
		return target
	}
}

func generate() error {
	projection := map[string]any{
		"version":    "poc.context-projection/v1",
		"registryID": "simulated_contract_registry",
		"budget": map[string]any{
			"maxTurnStartTokens":      300,
			"maxUserPromptHintTokens": 200,
		},
		"fragments": map[string]any{
			"sim.internal_registry_context":     projected("sim.internal_registry_context", "sim.schema", "internal_model_context", "message", true, true, "message", "user", "POC_SENTINEL_INTERNAL_CONTEXT", "POC_SENTINEL_INTERNAL_CONTEXT: contract/ is the only active authority root."),
			"sim.skill_context":                 projected("sim.skill_context", "sim.schema", "available_skills", "message", true, true, "message", "developer", "POC_SENTINEL_AVAILABLE_SKILLS", "POC_SENTINEL_AVAILABLE_SKILLS: simulated skill context."),
			"sim.turn_start_additional_context": projected("sim.turn_start_additional_context", "sim.schema", "turn_start_additional_context", "message", true, false, "message", "developer", "POC_SENTINEL_ADDITIONAL_CONTEXT", "POC_SENTINEL_ADDITIONAL_CONTEXT: simulated client-provided application context."),
			"sim.prompt_hint":                   projected("sim.prompt_hint", "sim.resolver", "hook_prompt_fragment", "message", true, true, "message", "user", "POC_SENTINEL_HOOK_HINT", "POC_SENTINEL_HOOK_HINT: selectedFragments=[sim.internal_registry_context]."),
			"sim.json_tool_result":              projected("sim.json_tool_result", "sim.resolver", "json_tool_output", "tool_output", false, true, "function_call_output", "", "POC_SENTINEL_JSON_TOOL_OUTPUT", "POC_SENTINEL_JSON_TOOL_OUTPUT"),
			"sim.mcp_tool_result":               projected("sim.mcp_tool_result", "sim.resolver", "mcp_tool_output", "tool_output", false, true, "function_call_output", "", "POC_SENTINEL_MCP_TOOL_OUTPUT", "POC_SENTINEL_MCP_TOOL_OUTPUT"),
		},
	}

	proofCases := proofCasesFromProjection(mapFromProjection(projection), true)

	report := map[string]any{
		"version": "poc.runtime-proof-report/v1",
		"upstream": map[string]any{
			"repo": "openai/codex",
			"ref":  upstreamRef,
		},
		"cases": proofCases,
		"pass":  true,
	}

	hints := map[string]any{
		"version":           "resolver.user-prompt-submit-hints/v1",
		"selectedFragments": []string{"sim.internal_registry_context"},
		"hints": []map[string]any{
			{
				"id":         "hint.sim.prompt_hint",
				"kind":       "fragment-selection",
				"fragmentID": "sim.internal_registry_context",
				"reason":     "Simulated prompt selects the registry context fragment.",
				"confidence": "high",
				"policy":     map[string]any{"toolExposure": "deny"},
			},
		},
		"sentinel": "POC_SENTINEL_HOOK_HINT",
	}

	routes := []map[string]any{
		promptRouteFixture("architecture", "architecture", []string{"sim.internal_registry_context", "sim.prompt_hint"}, true, 0.72, "Validate the resolver boundary."),
		promptRouteFixture("dotfiles-lua", "dotfiles", []string{"sim.skill_context"}, false, 0.94, "Route a Lua dotfiles task."),
		promptRouteFixture("simple-issue", "issue-generation", []string{"sim.prompt_hint"}, false, 0.97, "Generate a scoped issue."),
		promptRouteFixture("complex-resolver-design", "resolver-design", []string{"sim.internal_registry_context", "sim.skill_context", "sim.prompt_hint"}, true, 0.68, "Design a complex resolver change."),
	}

	validDerivation := map[string]any{
		"routeID":           "complex-resolver-design",
		"advisoryOnly":      true,
		"source":            "sdk-resolver",
		"summary":           "Preserve native registry delivery while adding vetted prompt-time hints.",
		"selectedFragments": []string{"sim.internal_registry_context", "sim.prompt_hint"},
		"hints": map[string]any{
			"objective":            "Validate prompt-time selection and advisory derivation.",
			"constraints":          []string{"Do not emit the full registry.", "Do not treat tool output as implied context."},
			"suggestedNextActions": []string{"Vet selected fragments.", "Inject only compact additionalContext."},
			"forbiddenAssumptions": []string{"MCP output is native model context."},
		},
		"validity": map[string]any{"cueValidated": true},
	}
	derivationCases := []map[string]any{
		{
			"id": "complex-valid", "routeID": "complex-resolver-design", "outputValid": true, "accepted": true, "fallback": "none",
			"injection":  map[string]any{"channel": "message", "itemKind": "message", "nativeContextInjection": true, "toolResultNativeContextInjection": false},
			"derivation": validDerivation,
		},
		{
			"id": "architecture-invalid", "routeID": "architecture", "outputValid": false, "accepted": false, "fallback": "route-only",
			"injection": map[string]any{"channel": "none", "itemKind": "none", "nativeContextInjection": false, "toolResultNativeContextInjection": false},
		},
	}

	stage2Report := map[string]any{
		"version": "poc.stage2-proof-report/v1",
		"proofs": []map[string]any{
			{"id": "prompt_route_declared_fragments", "pass": true},
			{"id": "prompt_route_no_registry_delivery", "pass": true},
			{"id": "prompt_route_additional_context_is_message", "pass": true},
			{"id": "sdk_derivation_optional", "pass": true},
			{"id": "sdk_derivation_schema_valid", "pass": true},
			{"id": "sdk_derivation_advisory_only", "pass": true},
			{"id": "sdk_derivation_compact_hint_only", "pass": true},
			{"id": "sdk_derivation_not_tool_context", "pass": true},
			{"id": "mcp_registry_resource_rejected_as_context", "pass": true},
		},
		"pass": true,
	}

	if err := writeJSON("generated/context_projection.json", projection); err != nil {
		return err
	}
	if err := writeJSON("generated/proof_cases.json", proofCases); err != nil {
		return err
	}
	if err := writeJSON("generated/hook_prompt_hints.json", hints); err != nil {
		return err
	}
	if err := writeJSON("generated/prompt_routes.json", routes); err != nil {
		return err
	}
	if err := writeJSON("generated/prompt_derivation_cases.json", derivationCases); err != nil {
		return err
	}
	if err := writeJSON("generated/stage2_expected_report.json", stage2Report); err != nil {
		return err
	}
	if err := writeJSON("testdata/expected_report.json", report); err != nil {
		return err
	}
	return validate()
}

func promptRouteFixture(id, class string, selected []string, requiresDerivation bool, confidence float64, objective string) map[string]any {
	return map[string]any{
		"id": id, "class": class, "selectedFragments": selected,
		"requiresDerivation": requiresDerivation, "confidence": confidence,
		"hints": map[string]any{
			"objective":   objective,
			"constraints": []string{"Use only declared fragments."},
			"nextActions": []string{"Apply the selected route."},
		},
		"emitsFullRegistry": false,
		"additionalContext": map[string]any{
			"channel": "message", "itemKind": "message", "nativeContextInjection": true,
		},
	}
}

func mapFromProjection(projection map[string]any) map[string]projectedFragment {
	rawFragments, _ := projection["fragments"].(map[string]any)
	frags := make(map[string]projectedFragment, len(rawFragments))
	for id, raw := range rawFragments {
		b, _ := json.Marshal(raw)
		var frag projectedFragment
		_ = json.Unmarshal(b, &frag)
		frags[id] = frag
	}
	return frags
}

func projected(id, sourceID, target, channel string, native, proofRequired bool, itemKind, role, sentinel, body string) map[string]any {
	m := map[string]any{
		"id":                             id,
		"sourceID":                       sourceID,
		"target":                         target,
		"expectedChannel":                channel,
		"expectedNativeContextInjection": native,
		"proofRequired":                  proofRequired,
		"expectedItemKind":               itemKind,
		"sentinel":                       sentinel,
		"renderedBody":                   body,
	}
	if role != "" {
		m["expectedRole"] = role
	}
	return m
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func patchUpstream() error {
	upstream := filepath.Join("upstream", "openai-codex")
	if _, err := os.Stat(filepath.Join(upstream, ".git")); err != nil {
		return errors.New("missing upstream/openai-codex checkout; clone openai/codex at the pinned ref first")
	}
	patchDir := filepath.Join("patches", "openai-codex")
	entries, err := os.ReadDir(patchDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".patch") {
			continue
		}
		patchPath, _ := filepath.Abs(filepath.Join(patchDir, entry.Name()))
		cmd := exec.Command("git", "apply", patchPath)
		cmd.Dir = upstream
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git apply %s failed: %w", entry.Name(), err)
		}
	}
	fmt.Println("patches applied")
	return nil
}

func testUpstream() error {
	upstream := filepath.Join("upstream", "openai-codex")
	if _, err := os.Stat(filepath.Join(upstream, ".git")); err != nil {
		return errors.New("missing upstream/openai-codex checkout")
	}
	cmd := exec.Command("cargo", "test", "-p", "codex-core", "context_injection_surfaces", "--", "--nocapture")
	cmd.Dir = filepath.Join(upstream, "codex-rs")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
