build:
	go build -o doppler main.go

release:
	doppler run -- ./scripts/pre-release.sh $(V)
	doppler run -- ./scripts/release.sh
