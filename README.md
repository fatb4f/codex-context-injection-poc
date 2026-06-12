# codex-context-injection-poc

Runtime validation POC for Codex context injection surfaces.

## Purpose

Prove, with runtime item-shape assertions, which upstream Codex interfaces provide native model-input context injection and which only produce tool/resource output.

```text
simulated registry
  -> CUE validates projection
  -> generator emits fixtures
  -> Codex runtime tests consume fixtures
  -> tests produce boolean proof report
```

Native context injection is defined as:

```text
ResponseItem::Message / ResponseInputItem::Message
  with ContentItem::InputText containing a sentinel
```

Tool output is not native context injection:

```text
ResponseInputItem::FunctionCallOutput
ResponseInputItem::CustomToolCallOutput
ResponseInputItem::McpToolCallOutput
```

In this POC, the generated proof treats both JSON tool output and MCP tool output as `tool_output` channel surfaces, with `function_call_output` as the runtime item kind asserted by the upstream patch. The important invariant is that neither surface is native model-input context.

## Upstream pin

```text
openai/codex @ eddc5c75ed527a8348bfcaa85692e53189600833
```

This POC is intentionally isolated from production `contract.cuemod` and dotfiles. It uses a simulated registry to prove interface semantics before binding a production registry.

## Repo layout

```text
contract/
  registry.cue                 # registry schema
  projection.cue               # projected fragment schema
  proof.cue                    # runtime proof report schema
  fixtures/simulated_registry.cue

generated/
  context_projection.json      # deterministic projection fixture
  hook_prompt_hints.json       # prompt-time hint fixture
  proof_cases.json             # expected runtime item-shape cases

cmd/poc/
  main.go                      # small Go adapter CLI

patches/openai-codex/
  0001-add-context-injection-surface-tests.patch

testdata/
  expected_report.json

upstream/
  .gitkeep                     # place openai/codex checkout here, or use a submodule
```

## Commands

```bash
just validate
just generate
just patch-upstream
just test-upstream
```

Equivalent Go entrypoints:

```bash
go run ./cmd/poc validate
go run ./cmd/poc generate
go run ./cmd/poc patch-upstream
go run ./cmd/poc test-upstream
```

## Runtime proof contract

Each case must emit or assert:

```json
{
  "id": "internal_model_context",
  "expectedNativeContextInjection": true,
  "observedItemKind": "message",
  "observedRole": "user",
  "containsSentinel": true,
  "pass": true
}
```

## Acceptance criteria

The POC passes only if:

1. Simulated registry validates through CUE.
2. Generated proof cases are deterministic.
3. Codex tests assert actual `ResponseItem` / `ResponseInputItem` variants.
4. Native context surfaces produce message items.
5. Tool-output surfaces do not produce message items.
6. Boolean proof report is emitted.
7. No production dotfiles or `contract.cuemod` paths are referenced.

## Important boundary

This POC proves the interface semantics:

```text
ContextualUserFragment / additional_context / HookPromptFragment
  -> native message context

JsonToolOutput / MCP CallToolResult
  -> tool output, not native context
```

It does not prove the final production registry layout.
