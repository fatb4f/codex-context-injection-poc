package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
		"testdata/expected_report.json",
	} {
		if err := validateJSON(path); err != nil {
			return err
		}
	}

	if _, err := exec.LookPath("cue"); err == nil {
		cmd := exec.Command("cue", "vet", "./contract/...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("cue vet failed: %w", err)
		}
	} else {
		return errors.New("cue not found")
	}

	fmt.Println("validation passed")
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

func generate() error {
	projection := map[string]any{
		"version":    "poc.context-projection/v1",
		"registryID": "simulated_contract_registry",
		"budget": map[string]any{
			"maxTurnStartTokens":      300,
			"maxUserPromptHintTokens": 200,
		},
		"fragments": map[string]any{
			"sim.internal_registry_context":     projected("sim.internal_registry_context", "sim.schema", "internal_model_context", true, "message", "user", "POC_SENTINEL_INTERNAL_CONTEXT", "POC_SENTINEL_INTERNAL_CONTEXT: contract/ is the only active authority root."),
			"sim.skill_context":                 projected("sim.skill_context", "sim.schema", "available_skills", true, "message", "developer", "POC_SENTINEL_AVAILABLE_SKILLS", "POC_SENTINEL_AVAILABLE_SKILLS: simulated skill context."),
			"sim.turn_start_additional_context": projected("sim.turn_start_additional_context", "sim.schema", "turn_start_additional_context", true, "message", "developer", "POC_SENTINEL_ADDITIONAL_CONTEXT", "POC_SENTINEL_ADDITIONAL_CONTEXT: simulated client-provided application context."),
			"sim.prompt_hint":                   projected("sim.prompt_hint", "sim.resolver", "hook_prompt_fragment", true, "message", "user", "POC_SENTINEL_HOOK_HINT", "POC_SENTINEL_HOOK_HINT: selectedFragments=[sim.internal_registry_context]."),
			"sim.json_tool_result":              projected("sim.json_tool_result", "sim.resolver", "json_tool_output", false, "function_call_output", "", "POC_SENTINEL_JSON_TOOL_OUTPUT", "POC_SENTINEL_JSON_TOOL_OUTPUT"),
			"sim.mcp_tool_result":               projected("sim.mcp_tool_result", "sim.resolver", "mcp_tool_output", false, "mcp_tool_call_output", "", "POC_SENTINEL_MCP_TOOL_OUTPUT", "POC_SENTINEL_MCP_TOOL_OUTPUT"),
		},
	}

	proofCases := []proofCase{
		{"internal_model_context", true, "message", "user", true, true},
		{"available_skills", true, "message", "developer", true, true},
		{"hook_prompt_fragment", true, "message", "user", true, true},
		{"json_tool_output", false, "function_call_output", "", true, true},
		{"mcp_tool_output", false, "mcp_tool_call_output", "", true, true},
	}

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

	if err := writeJSON("generated/context_projection.json", projection); err != nil {
		return err
	}
	if err := writeJSON("generated/proof_cases.json", proofCases); err != nil {
		return err
	}
	if err := writeJSON("generated/hook_prompt_hints.json", hints); err != nil {
		return err
	}
	if err := writeJSON("testdata/expected_report.json", report); err != nil {
		return err
	}
	return validate()
}

func projected(id, sourceID, target string, native bool, itemKind, role, sentinel, body string) map[string]any {
	m := map[string]any{
		"id":                             id,
		"sourceID":                       sourceID,
		"target":                         target,
		"expectedNativeContextInjection": native,
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
