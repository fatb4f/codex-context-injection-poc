package poc

#ContextProjection: {
	version: "poc.context-projection/v1"

	registryID: string

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

				expectedRole: {
					if f.target == "available_skills" {"developer"}
					if f.target == "internal_model_context" {"user"}
					if f.target == "turn_start_additional_context" {"developer"}
					if f.target == "hook_prompt_fragment" {"user"}
				}

				sentinel:     f.body =~ "(POC_SENTINEL_[A-Z0-9_]+)" // generator extracts exact sentinel
				renderedBody:  f.body
			}
		}
	}
}
