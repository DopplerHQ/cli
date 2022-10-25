.PHONY: build release test

build:
	go build -o doppler -ldflags="-X github.com/DopplerHQ/cli/pkg/version.ProgramVersion=dev-$(shell git rev-parse --abbrev-ref HEAD)-$(shell git rev-parse --short HEAD)" main.go

test:
	go test ./pkg/... -v

test-e2e:
	./tests/e2e.sh

test-packages:
	./tests/packages.sh

test-release:
	@if [ ! -f "$$GOPATH/bin/goreleaser" ]; then echo "Error: goreleaser is not installed\n\nYou can install goreleaser with 'go install github.com/goreleaser/goreleaser@latest'" && exit 1; fi
	$$GOPATH/bin/doppler run -- goreleaser release --snapshot --skip-publish --rm-dist --parallelism=4

scan:
	@if [ ! -f "$$GOPATH/bin/gosec" ]; then echo "Error: gosec is not installed\n\nYou can install gosec with 'go install github.com/securego/gosec/cmd/gosec@latest'\n" && exit 1; fi
	$$GOPATH/bin/gosec -quiet ./pkg/...
