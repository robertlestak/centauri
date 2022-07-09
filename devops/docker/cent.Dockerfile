FROM golang:1.18 as builder

WORKDIR /src

COPY . .

RUN go build -o /bin/cent cmd/cent/*.go

FROM debian:bullseye as runtime

COPY --from=builder /bin/cent /bin/cent

ENTRYPOINT [ "/bin/cent" ]