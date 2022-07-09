FROM golang:1.18 as builder

ARG VERSION

WORKDIR /src

COPY . .

RUN go build -ldflags="-X 'main.Version=$VERSION'" -o /bin/centaurid cmd/centaurid/*.go

FROM debian:bullseye as runtime

COPY --from=builder /bin/centaurid /bin/centaurid

ENTRYPOINT [ "/bin/centaurid" ]