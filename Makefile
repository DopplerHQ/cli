.PHONY: build release test

build:
	go build -o doppler -ldflags="-X github.com/DopplerHQ/cli/pkg/version.ProgramVersion=dev-$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)" main.go

release:
	./scripts/release/pre-release.sh $(v)
	doppler run -- ./scripts/release/release.sh
	doppler run -- ./scripts/release/post-release.sh

test:
	go test ./pkg/... -v

test-e2e:
	./tests.sh

test-release:
	goreleaser release --snapshot --skip-publish --skip-sign --rm-dist

scan:
	if [ ! -f "$$GOPATH/bin/gosec" ]; then echo "Error: gosec is not installed\n\nYou can install gosec with 'go get github.com/securego/gosec/cmd/gosec'\n" && exit 1; fi
	$$GOPATH/bin/gosec -quiet ./pkg/...
