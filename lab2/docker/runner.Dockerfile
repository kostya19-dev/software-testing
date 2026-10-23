FROM golang:1.26.1-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        curl jq python3 tcpdump ca-certificates openjdk-17-jre-headless npm \
    && rm -rf /var/lib/apt/lists/*

RUN go install gotest.tools/gotestsum@v1.13.0 \
    && go install github.com/go-delve/delve/cmd/dlv@v1.27.0 \
    && npm install -g allure-commandline@2.34.1

COPY scripts/container.sh /usr/local/bin/lab2-container
ENTRYPOINT ["bash", "/usr/local/bin/lab2-container"]
