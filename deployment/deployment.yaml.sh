#!/usr/bin/env bash

set -e

defaultSidecarInjectorImage="cyberark/sidecar-injector:latest"

usage() {
    cat <<EOF
Generate Kubernetes deployment manifest for sidecar-injector webhook service.

usage: ${0} [OPTIONS]

The following flags are available to configure the sidecar-injector deployment. They can only take on non-empty values.

       --sidecar-injector-image    Container image for the Sidecar Injector.
                                   (default: ${defaultSidecarInjectorImage})

       --secretless-image          Container image for the Secretless sidecar.

       --authenticator-image       Container image for the Kubernetes Authenticator sidecar.

       --secrets-provider-image    Container image for the Secrets Provider sidecar.

       --secrets-provider          Option to use secrets provider and use the conjur-connect config map

EOF
    exit 1
}

usage_if_empty() {
  if [[ -z "${1}" ]]
  then
      usage
  fi
}

# latest_supported_api_version returns the latest supported api version of a Kubernetes resource
function latest_supported_api_version() {
  local apiGroup
  apiGroup=$(kubectl api-resources | grep "${1}" | head -1 | awk '{print $3}')
  kubectl api-versions | grep "${apiGroup}" | cat
}

sidecarInjectorImage="${defaultSidecarInjectorImage}"

while [[ $# -gt 0 ]]; do
    case ${1} in
        --sidecar-injector-image)
            usage_if_empty "$2"
            sidecarInjectorImage="$2"
            shift
            ;;
        --secretless-image)
            usage_if_empty "$2"
            secretlessImageArg="- -secretless-image=${2}"
            shift
            ;;
        --authenticator-image)
            usage_if_empty "$2"
            authenticatorImageArg="- -authenticator-image=${2}"
            shift
            ;;
        --secrets-provider-image)
            usage_if_empty "$2"
            secretsProviderImageArg="- -secrets-provider-image=${2}"
            shift
            ;;
        --secrets-provider)
            secretsProvider="envFrom:
          - configMapRef:
              name: conjur-connect"
            ;;
        *)
            usage
            ;;
    esac
    shift
done

cat << EOL
apiVersion: $(latest_supported_api_version Deployment)
kind: Deployment
metadata:
  name: cyberark-sidecar-injector
  labels:
    app: cyberark-sidecar-injector
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cyberark-sidecar-injector
  template:
    metadata:
      labels:
        app: cyberark-sidecar-injector
    spec:
      containers:
        - name: cyberark-sidecar-injector
          image: ${sidecarInjectorImage}
          imagePullPolicy: Always
          args:
            - -tlsCertFile=/etc/webhook/certs/cert.pem
            - -tlsKeyFile=/etc/webhook/certs/key.pem
            - -port=8080
            ${authenticatorImageArg}
            ${secretlessImageArg}
            ${secretsProviderImageArg}
          ports:
            - containerPort: 8080
              name: https
          ${secretsProvider}
          volumeMounts:
            - name: certs
              mountPath: /etc/webhook/certs
              readOnly: true
      volumes:
        - name: certs
          secret:
            secretName: cyberark-sidecar-injector
EOL
