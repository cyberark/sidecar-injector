FROM golang:1.11-stretch as mutating-webhook-service-builder
ENV GO111MODULE=on

RUN mkdir -p /work
WORKDIR /work

ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

# sidecar-injector source files
COPY pkg ./pkg
COPY cmd ./cmd

ARG GIT_COMMIT_SHORT="dev"

# The `gitCommitShort` override is there to provide the git commit information in the final
# binary.
RUN go build \
    -ldflags="-X github.com/cyberark/sidecar-injector/pkg/version.gitCommitShort=$GIT_COMMIT_SHORT" \
    -o cyberark-sidecar-injector \
    ./cmd/sidecar-injector

FROM alpine:3.8

RUN apk add -u shadow libc6-compat curl openssl && \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl && \
    apk del curl && \
    # Add Limited user
    groupadd -r sidecar-injector \
             -g 777 && \
    useradd -c "sidecar-injector runner account" \
            -g sidecar-injector \
            -u 777 \
            -m \
            -r \
            sidecar-injector

USER sidecar-injector

COPY --from=mutating-webhook-service-builder \
     /work/cyberark-sidecar-injector \
     /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/cyberark-sidecar-injector"]
