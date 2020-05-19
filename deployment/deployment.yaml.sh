#!/usr/bin/env bash

set -e

usage() {
    cat <<EOF
Generate Kubernetes deployment manifest for sidecar-injector webhook service.

usage: ${0} [OPTIONS]

The following flags are available to configure the sidecar-injector deployment. They can only take on non-empty values.

       --sidecar-injector-image    Container image for the Sidecar Injector.
       --secretless-image          Container image for the Secretless sidecar.
       --authenticator-image       Container image for the Kubernetes Authenticator sidecar.
EOF
    exit 1
}

usage_if_empty() {
  if [[ -z "${1}" ]]
  then
      usage
  fi
}

sidecarInjectorImage="cyberark/sidecar-injector:latest"

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
        *)
            usage
            ;;
    esac
    shift
done

cat << EOL
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: cyberark-sidecar-injector
  labels:
    app: cyberark-sidecar-injector
spec:
  replicas: 1
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
          ports:
            - containerPort: 8080
              name: https
          volumeMounts:
            - name: certs
              mountPath: /etc/webhook/certs
              readOnly: true
      volumes:
        - name: certs
          secret:
            secretName: cyberark-sidecar-injector
EOL
