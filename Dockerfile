#=============== Sidecar Injector Build Container ===================
FROM golang:1.17-stretch as mutating-webhook-service-builder

ARG GIT_COMMIT_SHORT="dev"
ARG KUBECTL_VERSION=1.22.0

# On CyberArk dev laptops, golang module dependencies are downloaded with a
# corporate proxy in the middle. For these connections to succeed we need to
# configure the proxy CA certificate in build containers.
#
# To allow this script to also work on non-CyberArk laptops where the CA
# certificate is not available, we copy the (potentially empty) directory
# and update container certificates based on that, rather than rely on the
# CA file itself.
ADD build_ca_certificate /usr/local/share/ca-certificates/
RUN update-ca-certificates

RUN mkdir -p /work
WORKDIR /work

ENV GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0

# Download kubectl CLI
RUN curl -LO https://dl.k8s.io/release/v"${KUBECTL_VERSION}"/bin/linux/amd64/kubectl && \
    chmod +x kubectl

COPY go.mod go.sum ./
RUN go mod download

# sidecar-injector source files
COPY pkg ./pkg
COPY cmd ./cmd

# The `gitCommitShort` override is there to provide the git commit information in the final
# binary.
RUN go build \
    -ldflags="-X github.com/cyberark/sidecar-injector/pkg/version.gitCommitShort=$GIT_COMMIT_SHORT" \
    -o cyberark-sidecar-injector \
    ./cmd/sidecar-injector

#=============== Sidecar Injector Container =========================
FROM alpine:3.14

RUN apk add -u shadow libc6-compat curl openssl && \
    rm -rf /var/cache/apk/*

# Add Limited user
RUN groupadd -r sidecar-injector \
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
     /work/kubectl \
     /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/cyberark-sidecar-injector"]
