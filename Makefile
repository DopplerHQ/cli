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

scan:
	if [ ! -f "$$GOPATH/bin/gosec" ]; then echo "Error: gosec is not installed\n\nYou can install gosec with 'go get github.com/securego/gosec/cmd/gosec'\n" && exit 1; fi
	$$GOPATH/bin/gosec -quiet ./pkg/...
