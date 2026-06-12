package poc

#PromptDerivation: {
	routeID: string

	advisoryOnly: true

	source: "sdk-resolver" | "native-subagent"

	summary: string

	hints: {
		objective:            string
		constraints:          [...string]
		suggestedNextActions: [...string]
		forbiddenAssumptions: [...string]
	}

	selectedFragments: [...#FragmentID]

	validity: {
		cueValidated: true
	}
}

#PromptDerivationCase: {
	id: string

	routeID: string
	outputValid: bool
	accepted:    bool

	fallback: "none" | "route-only"

	injection: {
		channel:                          "message" | "none"
		itemKind:                         "message" | "none"
		nativeContextInjection:           bool
		toolResultNativeContextInjection: false
	}

	derivation?: #PromptDerivation
}

#PromptDerivationCases: [...#PromptDerivationCase]
