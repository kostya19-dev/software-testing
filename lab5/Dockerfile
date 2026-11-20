FROM golang:1.26-alpine AS build
WORKDIR /suite
COPY go.mod go.sum ./
RUN go mod download
COPY *_test.go ./
RUN go test -c -o /suite/auth.test .

FROM alpine:3.21
WORKDIR /suite
COPY --from=build /suite/auth.test .
COPY features ./features
ENTRYPOINT ["./auth.test", "-test.v", "-test.timeout=5m"]
