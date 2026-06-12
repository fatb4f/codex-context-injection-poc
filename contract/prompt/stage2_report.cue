package poc

#Stage2Proof: {
	id: string
	pass: true
}

#Stage2ExpectedReport: {
	version: "poc.stage2-proof-report/v1"
	proofs:  [...#Stage2Proof]
	pass:    true
}
