build:
	go build -o doppler main.go

release:
	printf "%s" $$(doppler secrets get GOOGLE_APPLICATION_CREDENTIALS_CONTENT --plain) > $$(doppler secrets get GOOGLE_APPLICATION_CREDENTIALS --plain)
	doppler run -- goreleaser release --rm-dist
	rm $(doppler secrets get GOOGLE_APPLICATION_CREDENTIALS --plain)
