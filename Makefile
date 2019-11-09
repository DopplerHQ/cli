build:
	go build -o doppler main.go

release:
	doppler run -- ./release.sh
