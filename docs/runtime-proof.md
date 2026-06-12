# Runtime proof model

## Native context injection

A surface passes native context injection only when the observed item kind is:

```text
message
```

and the message content contains the test sentinel.

## Tool output

A surface fails native context injection, intentionally, when the observed item kind is one of:

```text
function_call_output
custom_tool_call_output
mcp_tool_call_output
```

## Why final assistant text is not accepted as proof

Final assistant text may reflect model behavior, paraphrase, hallucination, or hidden prompt state. The POC proves interface behavior by inspecting runtime item variants before relying on model behavior.
