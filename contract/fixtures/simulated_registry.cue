package poc

#Registry: {
	version: "poc.registry/v1"
	id: "simulated_contract_registry"
	authorityRoot: "contract/"

	sources: {
		"sim.schema": {
			id: "sim.schema"
			kind: "schema"
			path: "contract/schema.cue"
			declared: true
		}

		"sim.resolver": {
			id: "sim.resolver"
			kind: "resolver"
			path: "contract/resolver/user_prompt_submit.cue"
			declared: true
		}

		"sim.tool_policy": {
			id: "sim.tool_policy"
			kind: "tool_policy"
			path: "contract/policy/tool_exposure.cue"
			declared: true
		}
	}

	fragments: {
		"sim.internal_registry_context": {
			id: "sim.internal_registry_context"
			sourceID: "sim.schema"
			target: "internal_model_context"
			injectAt: "turnStart"
			body: "POC_SENTINEL_INTERNAL_CONTEXT: contract/ is the only active authority root."
			policy: {
				authoritative: true
				toolExposure: "deny"
				allowUndeclaredPaths: false
			}
		}

		"sim.skill_context": {
			id: "sim.skill_context"
			sourceID: "sim.schema"
			target: "available_skills"
			injectAt: "turnStart"
			body: "POC_SENTINEL_AVAILABLE_SKILLS: simulated skill context."
			policy: {
				authoritative: false
				toolExposure: "deny"
				allowUndeclaredPaths: false
			}
		}

		"sim.turn_start_additional_context": {
			id: "sim.turn_start_additional_context"
			sourceID: "sim.schema"
			target: "turn_start_additional_context"
			injectAt: "turnStart"
			body: "POC_SENTINEL_ADDITIONAL_CONTEXT: simulated client-provided application context."
			policy: {
				authoritative: false
				toolExposure: "deny"
				allowUndeclaredPaths: false
			}
		}

		"sim.prompt_hint": {
			id: "sim.prompt_hint"
			sourceID: "sim.resolver"
			target: "hook_prompt_fragment"
			injectAt: "userPromptSubmit"
			body: "POC_SENTINEL_HOOK_HINT: selectedFragments=[sim.internal_registry_context]."
			policy: {
				authoritative: false
				toolExposure: "deny"
				allowUndeclaredPaths: false
			}
		}

		"sim.json_tool_result": {
			id: "sim.json_tool_result"
			sourceID: "sim.resolver"
			target: "json_tool_output"
			injectAt: "toolResult"
			body: "POC_SENTINEL_JSON_TOOL_OUTPUT"
			policy: {
				authoritative: false
				toolExposure: "deny"
				allowUndeclaredPaths: false
			}
		}

		"sim.mcp_tool_result": {
			id: "sim.mcp_tool_result"
			sourceID: "sim.resolver"
			target: "mcp_tool_output"
			injectAt: "toolResult"
			body: "POC_SENTINEL_MCP_TOOL_OUTPUT"
			policy: {
				authoritative: false
				toolExposure: "deny"
				allowUndeclaredPaths: false
			}
		}
	}

	constraints: [
		{
			id: "deny_undeclared_paths"
			text: "resolver cannot walk undeclared paths"
		},
		{
			id: "tool_exposure_deny_default"
			text: "tool exposure remains deny-by-default"
		},
		{
			id: "hook_hint_only"
			text: "UserPromptSubmit emits hints only"
		},
	]
}

registry: #Registry
