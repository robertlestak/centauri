FROM golang:1.18 as builder

WORKDIR /src

COPY . .

RUN go build -o /bin/centauri-agent cmd/centauri-agent/*.go

FROM debian:bullseye as runtime

COPY --from=builder /bin/centauri-agent /bin/centauri-agent

ENTRYPOINT [ "/bin/centauri-agent" ]