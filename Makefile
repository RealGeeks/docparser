
test:
	go test -race -cover

cover:
	go test github.com/RealGeeks/docparser -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: test cover
