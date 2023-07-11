FROM google/cloud-sdk:437.0.1-slim

ARG HELM_VERSION
ARG KUBECTL_VERSION=1.22.0

RUN mkdir -p /src
WORKDIR /src

# Install Helm client
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
RUN chmod 700 get_helm.sh
RUN ./get_helm.sh --no-sudo --version ${HELM_VERSION:-v3.7.0}

# Install Docker client
RUN apt-get update -y && \
    apt-get install -y apt-transport-https ca-certificates curl gnupg2 software-properties-common wget && \
    curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg | apt-key add - && \
    add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable" && \
    apt-get update && \
    apt-get install -y docker-ce && \
    rm -rf /var/lib/apt/lists/*

# Install kubectl CLI
RUN wget -O /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v"${KUBECTL_VERSION}"/bin/linux/amd64/kubectl && \
    chmod +x /usr/local/bin/kubectl

# Install kubectx and kubens
RUN wget -O /usr/local/bin/kubectx https://raw.githubusercontent.com/ahmetb/kubectx/master/kubectx && \
    chmod +x /usr/local/bin/kubectx
RUN wget -O /usr/local/bin/kubens https://raw.githubusercontent.com/ahmetb/kubectx/master/kubens && \
    chmod +x /usr/local/bin/kubens
