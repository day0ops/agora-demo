module github.com/day0ops/agora-demo/catalog-api

go 1.27.0

require (
	github.com/day0ops/agora-demo/fixtures v0.0.0
	github.com/day0ops/agora-demo/trust-scoring v0.0.0
)

replace github.com/day0ops/agora-demo/fixtures => ../fixtures

replace github.com/day0ops/agora-demo/trust-scoring => ../trust-scoring
