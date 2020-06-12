#!/usr/bin/env bash

set -e

defaultSidecarInjectorImage="cyberark/sidecar-injector:latest"
defaultDeploymentApiVersion="apps/v1"

usage() {
    cat <<EOF
Generate Kubernetes deployment manifest for sidecar-injector webhook service.

usage: ${0} [OPTIONS]

The following flags are available to configure the sidecar-injector deployment. They can only take on non-empty values.

       --sidecar-injector-image    Container image for the Sidecar Injector.
                                   (default: ${defaultSidecarInjectorImage})

       --secretless-image          Container image for the Secretless sidecar.

       --authenticator-image       Container image for the Kubernetes Authenticator sidecar.

       --deployment-api-version    The supported apiVersion for Deployments. This is the
                                   value that will be set in the Deployment manifest. It
                                   defaults to the supported apiVersion for Deployments on
                                   the latest Kubernetes release.

                                   (default: ${defaultDeploymentAPIVersion})

                                   If you're on an older version of Kubernetes, you can
                                   retrieve the supported apiVersion for Deployments by
                                   running the shell commands below.

                                   deploymentApiGroup=\$(kubectl api-resources | grep "deployments.*deploy.*Deployment" | awk '{print \$3}')
                                   deploymentApiVersion=\$(kubectl api-versions | grep "\${deploymentApiGroup}/")
                                   echo "\${deploymentApiVersion}"
EOF
    exit 1
}

usage_if_empty() {
  if [[ -z "${1}" ]]
  then
      usage
  fi
}

sidecarInjectorImage="${defaultSidecarInjectorImage}"
deploymentApiVersion="${defaultDeploymentAPIVersion}"

while [[ $# -gt 0 ]]; do
    case ${1} in
        --deployment-api-version)
            usage_if_empty "$2"
            deploymentApiVersion="$2"
            shift
            ;;
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
apiVersion: ${deploymentApiVersion}
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
