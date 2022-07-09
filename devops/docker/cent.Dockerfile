FROM golang:1.18 as builder

ARG VERSION

WORKDIR /src

COPY . .

RUN go build -ldflags="-X 'main.Version=$VERSION'" -o /bin/cent cmd/cent/*.go

FROM debian:bullseye as runtime

COPY --from=builder /bin/cent /bin/cent

ENTRYPOINT [ "/bin/cent" ]