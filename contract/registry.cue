package poc

#Registry: {
	version: "poc.registry/v1"

	id: string

	authorityRoot: "contract/"

	sources: [string]: #Source

	fragments: [string]: #Fragment

	constraints: [...#Constraint]
}

#Source: {
	id: string
	kind: "schema" | "projection" | "resolver" | "tool_policy"

	path: string
	declared: true
}

#Fragment: {
	id: string
	sourceID: string

	target:
		"internal_model_context" |
		"available_skills" |
		"turn_start_additional_context" |
		"hook_prompt_fragment" |
		"json_tool_output" |
		"mcp_tool_output"

	injectAt:
		"turnStart" |
		"userPromptSubmit" |
		"toolResult"

	body: string

	policy: {
		authoritative: bool | *false
		toolExposure: "deny" | *"deny"
		allowUndeclaredPaths: false
	}
}

#Constraint: {
	id: string
	text: string
}
