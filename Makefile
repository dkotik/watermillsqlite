default:
	(cd wmsqlitemodernc && go test -short -failfast ./...)
	(cd wmsqlitezombiezen && go test -short -failfast ./...)
test:
	(cd wmsqlitemodernc && go test -v -count=5 -failfast -timeout=18m ./...)
	(cd wmsqlitezombiezen && go test -v -count=5 -failfast -timeout=18m ./...)
test_race:
	(cd wmsqlitemodernc && go test -v -count=5 -failfast -timeout=18m -race ./...)
	(cd wmsqlitezombiezen && go test -v -count=5 -failfast -timeout=18m -race ./...)
benchmark:
	(cd wmsqlitemodernc && go test -bench=. -run=^BenchmarkAll$$ -timeout=15s)
	(cd wmsqlitezombiezen && go test -bench=. -run=^BenchmarkAll$$ -timeout=15s)
