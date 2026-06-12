package poc

#ContextProjection: {
	version: "poc.context-projection/v1"

	registryID: "simulated_contract_registry"

	fragments: [string]: #ProjectedFragment

	budget: {
		maxTurnStartTokens:      int & >0 | *300
		maxUserPromptHintTokens: int & >0 | *200
	}
}

#ProjectedFragment: {
	id: string

	sourceID: string

	target:
		"internal_model_context" |
		"available_skills" |
		"turn_start_additional_context" |
		"hook_prompt_fragment" |
		"json_tool_output" |
		"mcp_tool_output"

	expectedNativeContextInjection: bool

	expectedItemKind:
		"message" |
		"function_call_output" |
		"custom_tool_call_output" |
		"mcp_tool_call_output"

	expectedRole?: "user" | "developer"

	sentinel: =~"^POC_SENTINEL_[A-Z0-9_]+$"

	renderedBody: string
}

projection: #ContextProjection & {
	version: "poc.context-projection/v1"
	registryID: registry.id

	budget: {
		maxTurnStartTokens:      300
		maxUserPromptHintTokens: 200
	}

	fragments: {
		for id, f in registry.fragments {
			(id): #ProjectedFragment & {
				id:       f.id
				sourceID: f.sourceID
				target:   f.target

				expectedNativeContextInjection: f.target == "internal_model_context" ||
					f.target == "available_skills" ||
					f.target == "turn_start_additional_context" ||
					f.target == "hook_prompt_fragment"

				expectedItemKind: {
					if f.target == "json_tool_output" {
						"function_call_output"
					}
					if f.target == "mcp_tool_output" {
						"mcp_tool_call_output"
					}
					if f.target != "json_tool_output" && f.target != "mcp_tool_output" {
						"message"
					}
				}

				if f.target == "available_skills" {
					expectedRole: "developer"
					sentinel: "POC_SENTINEL_AVAILABLE_SKILLS"
				}
				if f.target == "internal_model_context" {
					expectedRole: "user"
					sentinel: "POC_SENTINEL_INTERNAL_CONTEXT"
				}
				if f.target == "turn_start_additional_context" {
					expectedRole: "developer"
					sentinel: "POC_SENTINEL_ADDITIONAL_CONTEXT"
				}
				if f.target == "hook_prompt_fragment" {
					expectedRole: "user"
					sentinel: "POC_SENTINEL_HOOK_HINT"
				}
				if f.target == "json_tool_output" {
					sentinel: "POC_SENTINEL_JSON_TOOL_OUTPUT"
				}
				if f.target == "mcp_tool_output" {
					sentinel: "POC_SENTINEL_MCP_TOOL_OUTPUT"
				}

				renderedBody:  f.body
			}
		}
	}
}
