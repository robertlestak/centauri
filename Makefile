bin: bin/mp_darwin bin/mp_linux bin/mp_windows

bin/mp_darwin:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/mp_darwin cmd/mp/*.go

bin/mp_linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/mp_linux cmd/mp/*.go

bin/mp_windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/mp_windows cmd/mp/*.go