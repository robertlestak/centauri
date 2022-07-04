bin: bin/centauri_darwin bin/centauri_linux bin/centauri_windows

bin/centauri_darwin:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/centauri_darwin cmd/centauri/*.go

bin/centauri_linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/centauri_linux cmd/centauri/*.go

bin/centauri_windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/centauri_windows cmd/centauri/*.go