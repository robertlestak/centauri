VERSION=v0.0.1

binaries: centaurid cent centauri-agent

docker: docker-centaurid docker-cent docker-centauri-agent

centaurid: bin/centaurid_darwin bin/centaurid_windows bin/centaurid_linux
cent: bin/cent_darwin bin/cent_windows bin/cent_linux
centauri-agent: bin/centauri-agent_darwin bin/centauri-agent_windows bin/centauri-agent_linux

bin/centaurid_darwin:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/centaurid_darwin cmd/centaurid/*.go
	openssl sha512 bin/centaurid_darwin > bin/centaurid_darwin.sha512

bin/centaurid_linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/centaurid_linux cmd/centaurid/*.go
	openssl sha512 bin/centaurid_linux > bin/centaurid_linux.sha512

bin/centaurid_windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/centaurid_windows cmd/centaurid/*.go
	openssl sha512 bin/centaurid_windows > bin/centaurid_windows.sha512

bin/centauri-agent_darwin:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/centauri-agent_darwin cmd/centauri-agent/*.go
	openssl sha512 bin/centauri-agent_darwin > bin/centauri-agent_darwin.sha512

bin/centauri-agent_linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/centauri-agent_linux cmd/centauri-agent/*.go
	openssl sha512 bin/centauri-agent_linux > bin/centauri-agent_linux.sha512

bin/centauri-agent_windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/centauri-agent_windows cmd/centauri-agent/*.go
	openssl sha512 bin/centauri-agent_windows > bin/centauri-agent_windows.sha512

bin/cent_darwin:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/cent_darwin cmd/cent/*.go
	openssl sha512 bin/cent_darwin > bin/cent_darwin.sha512

bin/cent_linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/cent_linux cmd/cent/*.go
	openssl sha512 bin/cent_linux > bin/cent_linux.sha512

bin/cent_windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -ldflags="-X 'main.Version=$(VERSION)'" -o bin/cent_windows cmd/cent/*.go
	openssl sha512 bin/cent_windows > bin/cent_windows.sha512

docker-centaurid:
	docker build --build-arg VERSION=$(VERSION) -f devops/docker/centaurid.Dockerfile -t centaurid .
	docker tag centaurid:latest centaurid:$(VERSION)

docker-centauri-agent:
	docker build --build-arg VERSION=$(VERSION) -f devops/docker/centauri-agent.Dockerfile -t centauri-agent .
	docker tag centauri-agent:latest centauri-agent:$(VERSION)

docker-cent:
	docker build --build-arg VERSION=$(VERSION) -f devops/docker/cent.Dockerfile -t cent .
	docker tag cent:latest cent:$(VERSION)

.PHONY: docker binaries centaurid cent centauri-agent
.PHONY: docker-centaurid docker-centauri-agent docker-cent
