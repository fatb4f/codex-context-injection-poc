package poc

#RuntimeProofCase: {
	id: string

	input: {
		target: string
		sentinel: string
	}

	expected: {
		nativeContextInjection: bool
		itemKind: string
		role?: string
		containsSentinel: true
	}

	observed?: {
		nativeContextInjection: bool
		itemKind: string
		role?: string
		containsSentinel: bool
	}

	pass?: bool
}

#RuntimeProofReport: {
	version: "poc.runtime-proof-report/v1"

	upstream: {
		repo: "openai/codex"
		ref:  string
	}

	cases: [...#RuntimeProofCase]

	pass: bool & (len([for c in cases if c.pass == false {c}]) == 0)
}
