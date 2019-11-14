build:
	go build -o doppler main.go

release:
	./scripts/pre-release.sh $(v)
	doppler run -- ./scripts/release.sh
	doppler run -- ./scripts/post-release.sh

test-release:
	goreleaser release --snapshot --skip-publish --skip-sign --rm-dist
