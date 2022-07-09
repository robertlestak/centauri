FROM golang:1.18 as builder

WORKDIR /src

COPY . .

RUN go build -o /bin/centaurid cmd/centaurid/*.go

FROM debian:bullseye as runtime

COPY --from=builder /bin/centaurid /bin/centaurid

ENTRYPOINT [ "/bin/centaurid" ]