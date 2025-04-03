# CyberArk Sidecar Injector

Sidecars are used in Kubernetes to introduce additional features to application pods.
Manual sidecar injection can be cumbersome and repetitive. **CyberArk Sidecar Injector**
enables automatic sidecar injection through it being a
[mutating admission webhook controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook).
This means when **CyberArk Sidecar Injector** is deployed and enabled in a namespace, any
pod created in that namespace with the appropriate annotations will result in automated
sidecar injection.

**CyberArk Sidecar Injector** provides support for a selection of available sidecars that
are configurable through annotations.

_Note that unlike manual injection, automatic injection occurs at the pod-level. You will
not see any change to the deployment itself. Instead you will want to check individual
pods (via kubectl describe) to see the injected sidecar._

***

##Status: Beta

The CyberArk Sidecar Injector is currently in beta.

Naming and functionality are still subject to **breaking** changes.

***
## Certification Level
![](https://img.shields.io/badge/Certification%20Level-Certified-28A745?link=https://github.com/cyberark/community/blob/master/Conjur/conventions/certification-levels.md)

The Secrets Provider push to File feature is a **Certified** level project.
Certified projects **have been reviewed and are supported by CyberArk**. For more
detailed information on our certification levels, see
[our community guidelines](https://github.com/cyberark/community/blob/master/Conjur/conventions/certification-levels.md#community).

***
This document shows how to deploy and use the CyberArk Sidecar Injector
[mutating admission webhook controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)
which injects sidecar container(s) into a pod prior to persistence of the underlying
object.

  * [Prerequisites](#prerequisites)
    + [Mandatory TLS](#mandatory-tls)
  * [Available Sidecars](#available-sidecars)
    + [Secretless](#secretless)
    + [Authenticator](#authenticator)
    + [Secrets Provider](#secrets-provider)
  * [Installation](#installation)
    + [Installing the Sidecar Injector (Manually)](#installing-the-sidecar-injector-manually)
      - [Dedicated Namespace](#dedicated-namespace)
      - [Deploy Sidecar Injector](#deploy-sidecar-injector)
      - [Verify Sidecar Injector Installation](#verify-sidecar-injector-installation)
    + [Installing the Sidecar Injector (Helm)](#installing-the-sidecar-injector-helm)
  * [Using the Sidecar Injector](#using-the-sidecar-injector)
    + [Configuration](#configuration)
      - [conjur.org/secretless-config](#conjurorgsecretlessconfig)
      - [conjur.org/conjurConnConfig](#conjurorgconjurconnconfig)
      - [conjur.org/conjurAuthConfig](#conjurorgconjurauthconfig)
  * [Secretless Sidecar Injection Example](#secretless-sidecar-injection-example)
  * [Conjur Authenticator/Secretless/Secrets Provider Sidecar Injection Example](#conjur-authenticatorsecretless-sidecarsecrets-provider-injection-example)
    + [Deploy Authenticator Sidecar](#deploy-authenticator-sidecar)
    + [Deploy Secretless Sidecar](#deploy-secretless-sidecar)
  * [License](#license)

## Prerequisites

Kubernetes 1.16.0 or above with the `admissionregistration.k8s.io/v1` API enabled.
Verify that by the following command:
```
~$ kubectl api-versions | grep admissionregistration.k8s.io/v1
```
The result should be:
```
admissionregistration.k8s.io/v1
```

In addition, the `MutatingAdmissionWebhook` and `ValidatingAdmissionWebhook` admission
controllers should be added and listed in the correct order in the admission-control flag
of kube-apiserver. Please see the [Kubernetes documentation](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/).
It is likely that this is set by default if your cluster is running on GKE.

If using `minikube`, start your cluster as follows:
```bash
~$ minikube start --kubernetes-version=v1.10.0
```

### Available Sidecars

This section enumerates the available sidecars. The choice of sidecar is made via
annotations - see the [Configuration](#configuration) section for details.

1. [Secretless](#secretless)
2. [Authenticator](#authenticator)
3. [Secrets Provider](#secrets-provider)

#### Secretless
See the [README](https://github.com/cyberark/secretless-broker)

Injects: 
  + a newly created **Volume** named `secretless-config` sourced from a ConfigMap
  + a configurable **Secretless container** with:
    + a **Volume Mount** at `/etc/secretless` of the `secretless-config` Volume

#### Authenticator
See the [README](https://github.com/cyberark/conjur-authn-k8s-client)

Injects:
  + a newly created shareable **in-memory Volume** named `conjur-access-token` used to
  store the access token
  + a configurable **Authenticator container** with:
    + a **Volume Mount** at `/run/conjur` of the `conjur-access-token` Volume
  + a read-only **Volume Mount** at `/run/conjur` of the `conjur-access-token` Volume for
    each of the containers whose names appears in the comma-separated list of container
    names that you may configure.

#### Secrets Provider
See the [README](https://github.com/cyberark/secrets-provider-for-k8s)

Injects:
  + Newly created shareable **in-memory Volumes** named `podInfo` and `conjur-secrets`.
  + a configurable **[Secrets Provider container](https://github.com/cyberark/secrets-provider-for-k8s)**
  + **Volume Mounts** at `/conjur/secrets` and `/conjur/podinfo`

Requires:
  + a configMap in the namespace that the sidecar resides in. The config map can be the Golden config or the Conjur connect config map.
  + The sidecar deployment manifest must include the conjur-connect configmap or the Golden configmap

For example to add the Golden config to the sidecar manifest:
```
    envFrom:
      - configMapRef:
          name: conjur-configmap
```

### Mandatory TLS

Supporting TLS for external webhook server is required because admission is a high
security operation. As part of the installation process, we need to create a TLS
certificate signed by a trusted CA (shown below is the Kubernetes CA but you can use your
own) to secure the communication between the webhook server and kube-apiserver. For the
complete steps of creating and approving Certificate Signing Requests(CSR), please refer
to
[Managing TLS in a cluster](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/).


## Docker Image

The docker image for the mutating admission webhook server is publicly available on
Dockerhub as
[cyberark/sidecar-injector](https://hub.docker.com/r/cyberark/sidecar-injector/).

The docker image entrypoint is the server binary. The binary supports the following flags:
```bash
Usage of cyberark-sidecar-injector:
  -authenticator-image string
        Container image for the Kubernetes Authenticator sidecar (default "cyberark/conjur-authn-k8s-client:latest")
  -noHTTPS
        Run Webhook server as HTTP (not HTTPS).
  -port int
        Webhook server port. (default 443)
  -secretless-image string
        Container image for the Secretless sidecar (default "cyberark/secretless-broker:latest")
  -tlsCertFile string
        Path to file containing the x509 Certificate for HTTPS. (default "/etc/webhook/certs/cert.pem")
  -tlsKeyFile string
        Path to file containing the x509 Private Key for HTTPS. (default "/etc/webhook/certs/key.pem")
  -version
        Show current version
```

## Installation

Installation is possible either
+ [manually](#installing-the-sidecar-injector-manually) or
+ using a [Helm chart](#installing-the-sidecar-injector-helm)


### Installing the Sidecar Injector (Manually)

#### Dedicated Namespace

Create a namespace `injectors`, where you will deploy the CyberArk Sidecar Injector
Webhook components.

1. Create namespace
    ```bash
    ~$ kubectl create namespace injectors
    ```

#### Deploy Sidecar Injector

1. Create a signed cert/key pair and store it in a Kubernetes `secret` that will be
consumed by sidecar injector deployment
    ```bash
    ~$ ./deployment/webhook-create-signed-cert.sh \
        --service cyberark-sidecar-injector \
        --secret cyberark-sidecar-injector \
        --namespace injectors
    ```

2. Patch the `MutatingWebhookConfiguration` by setting `caBundle` with correct value from
Kubernetes cluster
    ```bash
    ~$ cat deployment/mutatingwebhook.yaml | \
        deployment/webhook-patch-ca-bundle.sh \
          --namespace-selector-label cyberark-sidecar-injector \
          --service cyberark-sidecar-injector \
          --namespace injectors > \
        deployment/mutatingwebhook-ca-bundle.yaml
    ```

3. Generate sidecar injector deployment manifest.
Note: add `--secrets-provider ` if using secrets provider.
    ```bash
    ~$ ./deployment/deployment.yaml.sh \
         --deployment-api-version apps/v1 \
         --sidecar-injector-image cyberark/sidecar-injector:latest \
         --secretless-image cyberark/secretless-broker:latest \
         --authenticator-image cyberark/conjur-authn-k8s-client:latest \
         --secrets-provider-image cyberark/secrets-provider-for-k8s:latest
         > ./deployment/deployment.yaml
    ```

4. Deploy resources
    ```bash
    ~$ kubectl -n injectors apply -f deployment/deployment.yaml
    ~$ kubectl -n injectors apply -f deployment/service.yaml
    ~$ kubectl -n injectors apply -f deployment/mutatingwebhook-ca-bundle.yaml
    ~$ kubectl -n injectors apply -f deployment/crd.yaml
    ```

#### Verify Sidecar Injector Installation

1. The sidecar injector webhook should be running
    ```bash
    ~$ kubectl -n injectors get pods
    ```
    ```
    NAME                                                  READY     STATUS    RESTARTS   AGE
    cyberark-sidecar-injector-bbb689d69-882dd   1/1       Running   0          5m
    ```
    ```bash
    ~$ kubectl -n injectors get deployment
    ```
    ```
    NAME                                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
    cyberark-sidecar-injector             1         1         1            1           5m
    ```

### Installing the Sidecar Injector (Helm)

+ [Helm v3](https://helm.sh/docs/) is **required**

To install the sidecar injector in the `injectors` namespace run the following:

```
helm --namespace injectors \
 install \
 --set "deploymentApiVersion=apps/v1" \
 --set "caBundle=$(kubectl -n kube-system \
   get configmap \
   extension-apiserver-authentication \
   -o=jsonpath='{.data.client-ca-file}' \
 )" \
 ./helm/cyberark-sidecar-injector/  --generate-name
```

Optionally, if you want to specify the Secretless, Conjur Authenticator and/or 
Secrets Provider Docker image references, you can specify this in the `helm install` command:
```
helm --namespace injectors \
 install \
 --set "deploymentApiVersion=apps/v1" \
 --set "caBundle=$(kubectl -n kube-system \
    get configmap \
    extension-apiserver-authentication \
    -o=jsonpath='{.data.client-ca-file}' \
  )" \
 --set secretlessImage=path/to/secretless/container/image/repo/and/tag" \
 --set authenticatorImage=path/to/authenticator/container/image/repo/and/tag" \
 --set secretsProviderImage=path/to/secretsProvider/container/image/repo/and/tag" \
 ./helm/cyberark-sidecar-injector/  --generate-name
```

Below is example output from the `helm install` command:
```
...

NOTES:
## Instructions
Before you can proceed to use the sidecar-injector, there's one last step.
You will need to approve the CSR (Certificate Signing Request) made by the
sidecar-injector.
This allows the sidecar-injector to communicate securely with the Kubernetes API.

### Watch initContainer logs for when CSR is created
kubectl -n injectors logs deployment/cyberark-sidecar-injector -c
init-webhook -f

### You can check and inspect the CSR
kubectl describe csr "cyberark-sidecar-injector.injectors"

### Approve the CSR
kubectl certificate approve "cyberark-sidecar-injector.injectors"

Now that everything is setup you can enjoy the Cyberark Sidecar Injector.
This is the general workflow:

1. Annotate your application to enable the injector and to configure the sidecar (see
README.md)
2. Webhook intercepts and injects containers as needed

Enjoy.

```

Make sure to read the NOTES section once the chart is installed; instructions are provided
on how to accept the CSR request. The CSR request **must** be approved.

## Using the Sidecar Injector

### Configuration

The configurable parameters should be set as annotations on the **Pod template spec** and
*NOT on the Deployment, Job or otherwise**. The sidecar injector will not inject the
sidecar into pods by default. Add the `conjur.org/inject` annotation
with value `true` to the **Pod template spec** to enable injection. Injection will not go
ahead if the annotations are not on the **Pod template spec**.

**Note:** Configurable parameters of the Sidecar Injector have changed!

For Sidecar injector versions 0.1.1 and below the parameters used `sidecar-injector.cyberark.com/xxx`.
This has been updated to `conjur.org/xxx` in version 0.2.0 and above.


The following table lists the configurable parameters of the Sidecar Injector and their
default values.

| Parameter                     | Description                                     | Default                                                    |
| -----------------------       | ---------------------------------------------   | ---------------------------------------------------------- |
| `conjur.org/inject`| Enable the Sidecar Injector by setting to `true`            | `nil` (required) |
| `conjur.org/secretless-config` | ConfigMap holding Secretless configuration               |  `nil` (required for secretless)  |
| `conjur.org/conjurAuthConfig` | ConfigMap holding Conjur authentication configuration            |  `nil` (required for authenticator |
| `conjur.org/conjurConnConfig` | ConfigMap holding Conjur connection configuration               |  `nil` (required for authenticator |
| `conjur.org/inject-type` | Injected Sidecar type (`secretless`, `authenticator` or `secrets-provider`)                    |  `nil` (required) |
| `conjur.org/conjur-inject-volumes` | Comma-separated list of the names of containers, in the pod, that will be injected with `conjur-access-token` or `conjur-secrets` and `conjur-status` VolumeMounts. (e.g. `app-container-1,app-container-2`)                  |  `nil` (applies to authenticator and secrets provider) |
| `conjur.org/container-mode` | Sidecar Container mode (`init` or `sidecar`)                  | (secretless only supports sidecar) defaults to `sidecar` |
| `conjur.org/container-name` | Sidecar Container name                  |  `nil` (only applies to authenticator and secrets-provider)                              |
| `conjur.org/container-image` | Sidecar Container image      | defaults to the value configured for the sidecar-injector at startup, using the `-secretless-image` or `-authenticator-image` or `-secrets-provider` CLI arguments. |

#### conjur.org/secretless-config

There are three options for the value of secretless-config:
  1. configmapName
	1. configfile#configmapname
	1. k8s/crd#crdName

Option one is legacy, providing backwards compatibility by only specifying a configmap
name. The general format is provider#identifier.

If a config map is referenced, it should contain the following path:
+ secretless.yml - Secretless Configuration File

For help using a CRD to configure secretless, Refer to the [secretless CRD readme](
https://github.com/cyberark/secretless-broker/tree/main/resource-definitions).

#### conjur.org/conjurConnConfig

Expected to contain the following paths:

+ CONJUR_VERSION - the version of your Conjur instance (4 or 5)
+ CONJUR_APPLIANCE_URL - the URL of the Conjur appliance instance you are connecting to
+ CONJUR_AUTHN_URL - the URL of the authenticator service endpoint
+ CONJUR_ACCOUNT - the account name for the Conjur instance you are connecting to
+ CONJUR_SSL_CERTIFICATE - the x509 certificate that was created when Conjur was initiated

#### conjur.org/conjurAuthConfig

Expected to contain the following path:

+ CONJUR_AUTHN_LOGIN - Host login for pod e.g.
namespace/service_account/some_service_account

## Secretless Sidecar Injection Example

For this section, you'll work from a test namespace `$TEST_APP_NAMESPACE_NAME` (see
below). Later you will label this namespace with `cyberark-sidecar-injector=enabled` so as
to allow the cyberark-sidecar-injector to operate on pods created in this namespace.

1. Set test namespace environment variable

    ```bash
    export TEST_APP_NAMESPACE_NAME=secretless-sidecar-test 
   ```
2. Create test namespace
    ```bash
    ~$ kubectl create namespace ${TEST_APP_NAMESPACE_NAME}
    ```

3. Label the default namespace with `cyberark-sidecar-injector=enabled`
    ```bash
    ~$ kubectl label \
      namespace ${TEST_APP_NAMESPACE_NAME} \
      cyberark-sidecar-injector=enabled

    ~$ kubectl get namespace \
      -L cyberark-sidecar-injector
    ```
    ```
    NAME                            STATUS    AGE       CYBERARK-SIDECAR-INJECTOR
    default                         Active    18h
    kube-public                     Active    18h
    kube-system                     Active    18h
    cyberark-sidecar-injector       Active    18h
    secretless-sidecar-test         Active    18h       enabled
    ```

4. Create Secretless ConfigMap

    This configuration sets up an `http` service authenticator listening on `0.0.0.0:3000`
    using the `basic_auth` authentication strategy. The service authenticator is passed
    the actual values for the user and password using the `literal` secret provider.
   
   As shown below, the username and password are the literal values `my-username` and
   `secretpassword`, respectively.

    ```bash
    ~$ cat << EOL | kubectl -n ${TEST_APP_NAMESPACE_NAME} create configmap secretless --from-file=secretless.yml=/dev/stdin
    version: "2"
    services:
      my-service-proxy:
        protocol: http
        listenOn: tcp://0.0.0.0:3000
        credentials:
          username: my-username
          password: secretpassword
        config:
          authenticationStrategy: basic_auth
          authenticateURLsMatching:
            - ^http.*
    EOL
    ```

5. Deploy an **echo server** app with the Secretless Sidecar:
   
   The app is an **echo server** listening on port **8080**, which echoes the request
   header of any requests sent to it.
   
   The **Secretless Sidecar** is injected into the application pod on pod creation via the
   sidecar injector. The injection is configured via annotations.
   
   + The `secretless` ConfigMap is used as a source for Secretless configuration.

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      delete pod \
      test-app --ignore-not-found

    ~$ cat << EOF | kubectl -n ${TEST_APP_NAMESPACE_NAME} create -f -
    apiVersion: v1
    kind: Pod
    metadata:
      name: test-app
      annotations:
        conjur.org/inject: "yes"
        conjur.org/secretless-config: "secretless"
        conjur.org/inject-type: "secretless"
      labels:
        app: test-app
    spec:
      containers:
        - name: app
          env:
            - name: http_proxy
              value: "http://0.0.0.0:3000"
          image: googlecontainer/echoserver:1.1
    EOF
    ```

6. Verify Secretless sidecar container injected
    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} get pods
    ```
    ```
    NAME                     READY     STATUS        RESTARTS   AGE
    test-app                 2/2       Running       0          1m
    ```

7. Test Secretless

    In this step, you test Secretless by `exec`ing into the application pod's main
    container and issuing an HTTP request against the echo server proxied by Secretless.
    
    The pod spec for the echo server sets the environment variable `http_proxy` within the
    application to the Secretless `http` service authenticator's address
    `http://0.0.0.0:3000`. This allows Secretless to inject HTTP Authorization headers
    when proxying the request, as per the Secretless configuration above.
    
    The HTTP Authorization headers are extracted and base64 decoded from the response to
    retrieve the username and password.

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      exec test-app \
      -c app \
      -i \
      -- \
      curl --silent localhost:8080 \
        | grep authorization \
        | sed -e s/^authorization=Basic\ // \
        | base64 --decode; echo
   
    ```
    ```
    my-username:secretpassword
    ```
    

## Conjur Authenticator/Secretless Sidecar/Secrets Provider Injection Example

For this section, you'll work from a test namespace `$TEST_APP_NAMESPACE_NAME` (see
below). Later you will label this namespace with `cyberark-sidecar-injector=enabled` so as
to allow the cyberark-sidecar-injector to operate on pods created in this namespace.

1. Setup a Conjur appliance running with the Kubernetes authenticator installed and
enabled. e.g. run `./start` in
[kubernetes-conjur-deploy](https://github.com/cyberark/kubernetes-conjur-deploy/)

1. Load Conjur policy to create a host for the service account
`$TEST_APP_SERVICE_ACCOUNT`. e.g. `test-app-secretless` is made available by walking
through [kubernetes-conjur-demo](https://github.com/conjurdemos/kubernetes-conjur-demo) up
to and including `./3_init_conjur_cert_authority.sh`

1. Set up environment variables, if not already set from
[kubernetes-conjur-demo](https://github.com/conjurdemos/kubernetes-conjur-demo)
    ```bash
    # REQUIRED values, identical to those used for
    # kubernetes-conjur-deploy and kubernetes-conjur-demo
    export CONJUR_VERSION=...
    export CONJUR_NAMESPACE_NAME=...
    export CONJUR_ACCOUNT=...
    export AUTHENTICATOR_ID=...
    export TEST_APP_NAMESPACE_NAME=...
    ```

1. Set up pod-specific environment variables - modify to match your environment
    ```bash
    # REQUIRED values
    export TEST_APP_SERVICE_ACCOUNT=test-app-secretless
    export containerMode=sidecar
    ```

1. Generate derived Conjur connection environment variables
    ```bash
    # derived values
    ## CONJUR_APPLIANCE_URL
    CONJUR_APPLIANCE_URL="https://conjur-follower.${CONJUR_NAMESPACE_NAME}.svc.cluster.local/api"
    ## CONJUR_AUTHN_URL
    CONJUR_AUTHN_URL="https://conjur-follower.${CONJUR_NAMESPACE_NAME}.svc.cluster.local/api/authn-k8s/${AUTHENTICATOR_ID}"
    ## CONJUR_AUTHN_LOGIN
    if [ ${CONJUR_VERSION} == '4' ]; then
      CONJUR_AUTHN_LOGIN=${TEST_APP_NAMESPACE_NAME}/service_account/${TEST_APP_SERVICE_ACCOUNT}
    else
      CONJUR_AUTHN_LOGIN=host/conjur/authn-k8s/${AUTHENTICATOR_ID}/apps/${TEST_APP_NAMESPACE_NAME}/service_account/${TEST_APP_SERVICE_ACCOUNT}
    fi
    ## CONJUR_SSL_CERTIFICATE
    ### get one of the follower pod names
    follower_pod_name=$(kubectl -n ${CONJUR_NAMESPACE_NAME} \
      get pods \
      -l role=follower --no-headers \
      | awk '{ print $1 }' | head -1);
  
    CONJUR_SSL_CERTIFICATE=$(kubectl -n ${CONJUR_NAMESPACE_NAME} \
      exec \
      $follower_pod_name \
      -- cat /opt/conjur/etc/ssl/conjur.pem)
    ```

1. Create test namespace
    ```bash
    ~$ kubectl create namespace ${TEST_APP_NAMESPACE_NAME}
    ```

1. Label the default namespace with `cyberark-sidecar-injector=enabled`
    ```bash
    ~$ kubectl label \
      namespace ${TEST_APP_NAMESPACE_NAME} \
      cyberark-sidecar-injector=enabled

    ~$ kubectl get namespace -L cyberark-sidecar-injector
    ```
    ```
    NAME                            STATUS    AGE       CYBERARK-SIDECAR-INJECTOR
    default                         Active    18h
    kube-public                     Active    18h
    kube-system                     Active    18h
    cyberark-sidecar-injector       Active    18h
    conjur-sidecar-test             Active    18h       enabled
    ```

1. Create service account (might already exist from `kubernetes-conjur-demo`) to be used
by the application pod
    
    This service account maps to the Conjur identity for the pod

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
    create serviceaccount ${TEST_APP_SERVICE_ACCOUNT}
    ```

1. Create Conjur ConfigMap

    This ConfigMap named `conjur` stores the connection details to the Conjur appliance. These
details are necessary for both the **Authenticator** and **Secretless** sidecars to
communicate with the Conjur appliance.
    ```bash
    ~$ cat << EOL | kubectl -n ${TEST_APP_NAMESPACE_NAME} apply -f -
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: conjur
    data:
      CONJUR_ACCOUNT: "${CONJUR_ACCOUNT}"
      CONJUR_VERSION: "${CONJUR_VERSION}"
      CONJUR_APPLIANCE_URL: "${CONJUR_APPLIANCE_URL}"
      CONJUR_AUTHN_URL: "${CONJUR_AUTHN_URL}"
      CONJUR_SSL_CERTIFICATE: |
    $(echo "${CONJUR_SSL_CERTIFICATE}" | awk '{ print "    " $0 }')
      CONJUR_AUTHN_LOGIN: "${CONJUR_AUTHN_LOGIN}"
    EOL
    ```

1. You can now leverage Conjur by either
   1. [Deploying the Authenticator Sidecar](#deploy-authenticator-sidecar) or
   2. [Deploying the Secretless Sidecar](#deploy-secretless-sidecar) or
   3. [Deploying the Secrets Provider Sidecar](#deploy-secrets-provider-sidecar)

### Deploy Authenticator Sidecar

1. Deploy an app with the Authenticator Sidecar:
    
    The **Authenticator Sidecar** is injected into the application pod on pod creation via the
sidecar injector. The injection is configured via annotations. 
    
    + The `conjur` ConfigMap is used for both Conjur Authentication and Connection configuration
    + The `conjur.org/container-name` is set to "secretless" because the
corresponding Conjur identity to the service account used expects the sidecar container to
be named secretless.

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      delete pod \
      test-app --ignore-not-found
   
    ~$ cat << EOF | kubectl -n ${TEST_APP_NAMESPACE_NAME} apply -f -
    apiVersion: v1
    kind: Pod
    metadata:
      annotations:
        conjur.org/conjurAuthConfig: conjur
        conjur.org/conjurConnConfig: conjur
        conjur.org/container-mode: ${containerMode}
        conjur.org/conjur-token-receivers: "app"
        conjur.org/inject: "yes"
        conjur.org/inject-type: authenticator
        conjur.org/container-name: secretless
      labels:
        app: test-app
      name: test-app
    spec:
      containers:
      - image: googlecontainer/echoserver:1.1
        name: app
      serviceAccountName: ${TEST_APP_SERVICE_ACCOUNT}
    EOF
    ```

1. Verify Authenticator sidecar container injected
    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} get pods
    ```
    ```
    NAME                     READY     STATUS        RESTARTS   AGE
    test-app                 2/2       Running       0          1m
    ```

1. Test Authenticator

    In this step, you test the Authenticator by `exec`ing into the application pod's main
    container and read the contents of `/run/conjur/access-token`.
    
    The `/run/conjur/access-token` file contains the access token which is injected by the
    **Authenticator** sidecar upon successful authentication against the Conjur appliance.
    Note that this file is volume mounted into the application pod's main container as a
    result of the annotation `conjur.org/conjur-token-receivers` being
    set to that container's name.

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      exec test-app \
      -c app \
      -i \
      -- \
      cat /run/conjur/access-token | jq .
    ```
    ```
    {
      "data": "host/conjur/authn-k8s/sidecar-test/apps/sidecar-example-app/service_account/test-app-secretless",
      "timestamp": "2018-09-20 16:54:04 UTC",
      "signature": "aICQDREA2S-ulOxu8yMWqT9o8h_JDKuuDKIJOFBbQsL_uKZuovManGn-q2Yr4wdT9f_kJdgCNsxh9q54w2ciptn5sAFB3YzDAmqfUzjWv9pIwel2o7N2nuzIw-h7Ho6hA2PQ8V1Iz3NSILCT2JAnWDTi_--bplxqa6g72-j0xprkuFMkDvj2cd084WtMMWXii4W_5WG6BWA9jtnd72-tzhoaU4LFSRfSK7LON8aDdzyFexkM1IbjIuiF1sASBIsvnuY2GeghNDO8VciKh6dXe-sBqNlISlYOTOaQoEMIxA8Nm2t9jeYxmDHJ0IFkTmneeC2dgaJWWoF7MtfJnyPvwn_Z-bF49hkcYDL37-xJxUHPDA4QoU_4p82oqgC3NPnI",
      "key": "11cd239ab55175a3c0f93a7376abe663"
    }
    ```


### Deploy Secretless Sidecar

1. Create Secretless ConfigMap:

   This configuration sets up an `http` service authenticator on `0.0.0.0:3000` using the
   `basic_auth` authentication strategy that retrieves user and password using the
   `conjur` secret provider.
   
   The username and password are set and stored within the Conjur appliance.
   
    ```bash
    ~$ cat << EOL | kubectl -n ${TEST_APP_NAMESPACE_NAME} create configmap secretless --from-file=secretless.yml=/dev/stdin
    services:
      my-service-proxy:
        protocol: http
        listenOn: tcp://0.0.0.0:3000
        credentials:
          username:
            from: conjur
            get: test-secretless-app-db/username
          password:
            from: conjur
            get: test-secretless-app-db/password
        config:
          authenticationStrategy: basic_auth
          authenticateURLsMatching:
            - ^http.*
    EOL
    ```

1. Deploy an **echo server** app with the Secretless Sidecar:

    The app is an **echo server** listening on port **8080**, which echoes the request
    header of any requests sent to it.
    
    The **Secretless Sidecar** is injected into the application pod on pod creation via
    the sidecar injector. The injection is configured via annotations.
    
    + The `conjur` ConfigMap is used for both Conjur Authentication and Connection
    configuration
    + The `secretless` ConfigMap is used as a source for Secretless configuration.
    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
    delete pod \
    test-app --ignore-not-found
    
    ~$ cat << EOF | kubectl -n ${TEST_APP_NAMESPACE_NAME} apply -f -
    apiVersion: v1
    kind: Pod
    metadata:
      annotations:
        conjur.org/conjurAuthConfig: conjur
        conjur.org/conjurConnConfig: conjur
        conjur.org/inject: "yes"
        conjur.org/inject-type: secretless
        conjur.org/secretless-config: secretless
      labels:
        app: test-app
      name: test-app
    spec:
      containers:
      - env:
          - name: http_proxy
            value: "http://0.0.0.0:3000"
        image: googlecontainer/echoserver:1.1
        name: app
      serviceAccountName: ${TEST_APP_SERVICE_ACCOUNT}
    EOF
    ```

1. Verify Secretless sidecar container injected
    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} get pods
    ```
    ```
    NAME                     READY     STATUS        RESTARTS   AGE
    test-app                 2/2       Running       0          1m
    ```

1. Test Secretless with Conjur
    
    In this step, you test Secretless by `exec`ing into the application pod's main
    container and issuing an HTTP request against the echo server proxied by Secretless.
    
    The pod spec for the echo server sets the environment variable `http_proxy` within the
    application to the Secretless `http` service authenticator's address
    `http://0.0.0.0:3000`. This allows Secretless to inject HTTP Authorization headers
    when proxying the request, as per the Secretless configuration above.
    
    The HTTP Authorization headers are extracted and base64 decoded from the response to
    retrieve the username and password.
    
    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      exec test-app \
      -c app \
      -i \
      -- \
      curl --silent localhost:8080 \
        | grep authorization \
        | sed -e s/^authorization=Basic\ // \
        | base64 --decode; echo
    ```
    ```
    "test_app:84674b2874a5d7c952e7fec8"
    ```
   
### Deploy Secrets Provider Sidecar

1. Deploy an app with the Secrets Provider Sidecar:

   The **Secrets Provider Sidecar** is injected into the application pod on pod creation via the
   sidecar injector. The injection is configured via annotations.

 + The `conjur.org/container-name` is set to "cyberark-secrets-provider-for-k8s".

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      delete pod \
      test-app --ignore-not-found
   
    ~$ cat << EOF | kubectl -n ${TEST_APP_NAMESPACE_NAME} apply -f -
    apiVersion: v1
    kind: Pod
    metadata:
      annotations:
        conjur.org/inject: "true"
        conjur.org/inject-type: "secrets-provider"
        conjur.org/container-name: "cyberark-secrets-provider-for-k8s"
        conjur.org/container-image: "docker.io/cyberark/secrets-provider-for-k8s:edge"
        conjur.org/conjur-token-receivers: "test-app"
        conjur.org/authn-identity: host/conjur/authn-k8s/my-authenticator-id/apps/test-app-secrets-provider-p2f
        conjur.org/container-mode: init
        conjur.org/secrets-destination: file
        conjur.org/conjur-secrets.test-app: |
          - admin-username: username
          - admin-password: password
        conjur.org/conjur-secrets-policy-path.test-app: test-secrets-provider-p2f-app-db/
        conjur.org/secret-file-path.test-app: "./application.yaml"
        conjur.org/secret-file-format.test-app: "yaml"
      labels:
        app: test-app
      name: test-app
    spec:
      containers:
      - image: googlecontainer/echoserver:1.1
        name: app
      serviceAccountName: ${TEST_APP_SERVICE_ACCOUNT}
    EOF
    ```

1. Verify Authenticator sidecar container injected
    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} get pods
    ```
    ```
    NAME                     READY     STATUS        RESTARTS   AGE
    test-app                 2/2       Running       0          1m
    ```

1. Test Secrets Provider

   In this step, you test the Secrets Provider by `exec`ing into the application pod's main
   container and read the file `/conjur/secrets/application.yaml`.

   The `/conjur/secrets/application.yaml` file contains the secrets that are injected by the
   **Secrets Provider** sidecar upon successful authentication against the Conjur appliance.
   Note that this file is volume mounted into the application pod's main container as a
   result of the annotation `conjur.org/conjur-inject-volumes` being
   set to that container's name.

    ```bash
    ~$ kubectl -n ${TEST_APP_NAMESPACE_NAME} \
      exec test-app \
      -c app \
      -i \
      -- \
      cat /conjur/secrets/application.yaml
    ```
    ```
    "admin-username": "test_app"
    "admin-password": "9622c67ebd1f5fa94ef114da"
    ```


## Contributing

We welcome contributions of all kinds to this repository. For instructions on how to get
started and descriptions of our development workflows, please see our [contributing
guide][contrib].

[contrib]: https://github.com/cyberark/sidecar-injector/blob/main/CONTRIBUTING.md

## License

The Sidecar Injector is licensed under Apache License 2.0 - see [`LICENSE`](LICENSE) for
more details.
