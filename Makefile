.PHONY: build release test

build:
	go build -o doppler main.go

release:
	./scripts/pre-release.sh $(v)
	doppler run -- ./scripts/release.sh
	doppler run -- ./scripts/post-release.sh

test:
	go test ./pkg/... -v

test-release:
	goreleaser release --snapshot --skip-publish --skip-sign --rm-dist

