FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/loadgen ./cmd/loadgen \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/exporter ./cmd/exporter

FROM scratch AS exporter
COPY --from=build /out/exporter /exporter
EXPOSE 9101
ENTRYPOINT ["/exporter"]

FROM scratch AS loadgen
COPY --from=build /out/loadgen /loadgen
EXPOSE 9102
ENTRYPOINT ["/loadgen"]
