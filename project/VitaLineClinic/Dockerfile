FROM golang:1.26-alpine AS build

WORKDIR /app/src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/clinic ./cmd/clinic

FROM alpine:3.21

WORKDIR /app

COPY --from=build /out/clinic /app/clinic
COPY scripts/docker-entrypoint.sh /app/docker-entrypoint.sh

RUN addgroup -S app && adduser -S -D -H -u 1000 -G app app \
    && chmod +x /app/docker-entrypoint.sh \
    && mkdir -p /app/config /app/logs \
    && chown -R app:app /app/config /app/logs

ENV VITALINE_LOG_FILE=/app/logs/clinic.log
VOLUME ["/app/logs"]

EXPOSE 8080
EXPOSE 8090

USER app

ENTRYPOINT ["/app/docker-entrypoint.sh", "/app/clinic"]
CMD ["api"]
