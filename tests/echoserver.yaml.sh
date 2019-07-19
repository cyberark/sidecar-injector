#!/usr/bin/env bash

: "${SECRETLESS_IMAGE:?"Need to set SECRETLESS_IMAGE non-empty, and available to cluster e.g. cyberark/secretless-broker:latest"}"

# returns current namespace if available, otherwise returns 'default'
current_namespace() {
  cur_ctx="$(kubectl config current-context)" || exit_err "error getting current context"
  ns="$(kubectl config view -o=jsonpath="{.contexts[?(@.name==\"${cur_ctx}\")].context.namespace}")" \
     || exit_err "error getting current namespace"

  if [[ -z "${ns}" ]]; then
    echo "default"
  else
    echo "${ns}"
  fi
}

# list of api groups that contain configurations
api_groups(){
  kubectl api-resources \
    |awk '/sbconfig/{print "  - "$3}'
}

cat << EOL
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: secretless-crd
rules:
- apiGroups:
  - apiextensions.k8s.io
  resources:
  - customresourcedefinitions
  verbs:
  - create
  - get
  - watch
  - list
- apiGroups: [""]
  resources:
  - namespaces
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - secretless${SECRETLESS_CRD_SUFFIX}.io
$(api_groups)
  resources:
  - configurations
  verbs:
  - get
  - list
  - watch

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: secretless-crd

---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: secretless-crd
subjects:
- kind: ServiceAccount
  name: secretless-crd
  namespace: $(current_namespace)
roleRef:
  kind: ClusterRole
  name: secretless-crd
  apiGroup: rbac.authorization.k8s.io

---
apiVersion: "secretless${SECRETLESS_CRD_SUFFIX}.io/v1"
kind: "Configuration"
metadata:
  name: crd-basic-auth-proxy
spec:
  listeners:
    - name: http_config_1_listener
      protocol: http
      address: 0.0.0.0:8000

  handlers:
    - name: http_config_1_handler
      type: basic_auth
      listener: http_config_1_listener
      match:
        - ^http.*
      credentials:
        - name: username
          provider: literal
          id: "username"
        - name: password
          provider: literal
          id: "password"

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sb-sci-echoserver
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sb-sci-echoserver
  template:
    metadata:
      labels:
        app: sb-sci-echoserver
      annotations:
        sidecar-injector.cyberark.com/inject: "yes"
        sidecar-injector.cyberark.com/secretlessConfig: "k8s/crd#crd-basic-auth-proxy"
        sidecar-injector.cyberark.com/injectType: "secretless"

    spec:
      serviceAccountName: secretless-crd
      containers:
      - name: echo-server
        image: gcr.io/google_containers/echoserver:1.10
        imagePullPolicy: Always
      # Secretless container to be added here by sidecar injector
EOL
