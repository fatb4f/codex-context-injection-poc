package poc

#FragmentID: =~"^[a-z0-9][a-z0-9._/-]*$"

#PromptRoute: {
	id:    string
	class: string

	selectedFragments: [...#FragmentID]

	requiresDerivation: *false | bool
	confidence:         number & >=0 & <=1

	hints?: {
		objective?:   string
		constraints?: [...string]
		nextActions?: [...string]
	}

	emitsFullRegistry: false

	additionalContext: {
		channel:                "message"
		itemKind:               "message"
		nativeContextInjection: true
	}
}

#PromptRoutes: [...#PromptRoute]
