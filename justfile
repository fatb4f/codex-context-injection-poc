set shell := ["bash", "-ceu"]

validate:
    go run ./cmd/poc validate

generate:
    go run ./cmd/poc generate

patch-upstream:
    go run ./cmd/poc patch-upstream

test-upstream:
    go run ./cmd/poc test-upstream

report:
    cat testdata/expected_report.json | jq .
