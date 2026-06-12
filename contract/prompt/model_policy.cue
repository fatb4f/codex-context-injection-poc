package poc

#AgentModelPolicy: {
	main: {
		model:           "gpt-5.5"
		reasoningEffort: "high"
	}

	promptResolver: {
		model:           "gpt-5.4-mini"
		reasoningEffort: "low"
		advisoryOnly:    true
		maxOutputTokens: int & >0 & <=2048
	}

	promptResolverEscalation: {
		model:           "gpt-5.5"
		reasoningEffort: "medium"
		allowedWhen: [
			"route.confidence < 0.75",
			"route.requiresDerivation == true",
			"schema_validation_failed_once",
		]
	}
}

agentModelPolicy: #AgentModelPolicy & {
	main: {
		model:           "gpt-5.5"
		reasoningEffort: "high"
	}
	promptResolver: {
		model:           "gpt-5.4-mini"
		reasoningEffort: "low"
		advisoryOnly:    true
		maxOutputTokens: 2048
	}
	promptResolverEscalation: {
		model:           "gpt-5.5"
		reasoningEffort: "medium"
		allowedWhen: [
			"route.confidence < 0.75",
			"route.requiresDerivation == true",
			"schema_validation_failed_once",
		]
	}
}
