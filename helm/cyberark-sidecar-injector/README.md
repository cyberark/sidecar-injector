# cyberark-sidecar-injector

CyberArk Sidecar Injector is a [MutatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#mutatingadmissionwebhook-beta-in-19) server which allows for configurable sidecar injection into a pod prior to persistence.

  * [TL;DR;](#tl-dr-)
  * [Introduction](#introduction)
  * [Prerequisites](#prerequisites)
    + [Mandatory TLS](#mandatory-tls)
  * [Installing the Chart](#installing-the-chart)
  * [Uninstalling the Chart](#uninstalling-the-chart)
  * [Configuration](#configuration)
    + [csrEnabled=true](#csrenabledtrue)
    + [certsSecret](#certssecret)

## TL;DR;

```bash
$ helm install -f values.yaml my-release .
```

## Introduction

This chart bootstraps a deployment of a CyberArk Sidecar Injector MutatingAdmissionWebhook server including the Service and MutatingWebhookConfiguration. 

## Prerequisites

- Kubernetes 1.4+ with Beta APIs enabled

### Mandatory TLS

Supporting TLS for external webhook server is required because admission is a high security operation. As part of the installation process, we need to create a TLS certificate signed by a trusted CA (shown below is the Kubernetes CA but you can use your own) to secure the communication between the webhook server and apiserver. For the complete steps of creating and approving Certificate Signing Requests(CSR), please refer to [Managing TLS in a cluster](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/).

## Installing the Chart

To install the chart with the release name `my-release`, follow the instructions in the NOTES section on how to approve the CSR:

```bash
$ helm install \
  --set caBundle="$(kubectl -n kube-system \
    get configmap \
    extension-apiserver-authentication \
    -o=jsonpath='{.data.client-ca-file}' \
  )" \
  my-release \ --generate-name
  .
```

```
...

NOTES:
## Instructions
Before you can proceed to use the sidecar-injector, there's one last step.
You will need to approve the CSR (Certificate Signing Request) made by the sidecar-injector.
This allows the sidecar-injector to communicate securely with the Kubernetes API.

### Watch initContainer logs for when CSR is created
kubectl -n injectors logs deployment/cyberark-sidecar-injector -c init-webhook -f

### You can check and inspect the CSR
kubectl describe csr "cyberark-sidecar-injector.injectors"

### Approve the CSR
kubectl certificate approve "cyberark-sidecar-injector.injectors"

Now that everything is setup you can enjoy the Cyberark Sidecar Injector.
This is the general workflow:

1. Annotate your application to enable the injector and to configure the sidecar (see README.md)
2. Webhook intercepts and injects containers as needed

Enjoy.

```

The command deploys the CyberArk Broker Sidecar Injector MutatingAdmissionWebhook on the Kubernetes cluster in the default configuration. In this configuration the chart uses the cluster CA certificate bundle with a Certificate Signing Request flow to allow TLS between the webhook server and the cluster. The caBundle is required. The [configuration](#configuration) section lists the parameters that can be configured during installation.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

The following table lists the configurable parameters of the CyberArk Sidecar Injector chart and their default values.

| Parameter | Description | Default|
| --- | --- | --- |
| `namespaceSelectorLabel` | Label which should be set to "enabled" for namespace to use Sidecar Injector | `cyberark-sidecar-injector` (required) |
| `caBundle`| CA certificate bundle that signs the server cert used by the webhook | `nil` (required) |
| `csrEnabled` | Generate a private key and certificate signing request towards the Kubernetes Cluster | `true` |
| `certsSecret` | Private key and signed certificate used by the webhook server | `nil` (required if csrEnabled is false) |
| `sidecarInjectorImage` | Container image for the sidecar injector. | `cyberark/sidecar-injector:latest` |
| `secretlessImage` | Container image for the Secretless sidecar. | `cyberark/secretless-broker:latest` |
| `authenticatorImage` | Container image for the Kubernetes Authenticator sidecar. | `cyberark/conjur-kubernetes-authenticator:latest` |
| `secretsProviderImage` | Container image for the Secrets Provider sidecar. | `cyberark/secrets-provider-for-k8s:latest` |
| `deploymentApiVersion` | The supported apiVersion for Deployments. This is the value that will be set in the Deployment manifest. It defaults to the supported apiVersion for Deployments on the latest Kubernetes release. | `apps/v1` |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```bash
$ helm install \
   --set csrEnabled="false" \
   --set certsSecret="some-secret" \
   --set caBundle="-----BEGIN CERTIFICATE-----..." \
   my-release \
   .
```

The above command creates a sidecar injector deployment, retrieves the private key and signed certificate from the `certsSecret` value and uses the `caBundle` value in the associated MutatingWebhookConfiguration. Note that `caBundle` is the certificate that signs the injector webhook server cert.

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example,

```bash
$ helm install -f values.yaml my-release .
```

### certsSecret

`certsSecret` is a Kubernetes Secret containing private key and signed certificate (on paths key.pem and cert.pem, respectively)
 used by the webhook server. 

It is required for the private key and signed certificate pair to contain entries for the DNS name of the webhook service, i.e., <service name>.<namespace>.svc, or the URL of the webhook server.

### caBundle

`caBundle` is the **required** CA certificate bundle that signs the server cert used by the webhook server. It is used in the MutatingWebhookConfiguration for the release.

### csrEnabled

When `csrEnabled` is set to `true`, the chart generate a private key and certificate signing request (CSR) towards the Kubernetes Cluster, and waits until the CSR is approved before deploying the sidecar injector. 

The private key and certificate will be stored in a secret created as part of the release.

The `caBundle` in this case is the Kubernetes cluster CA certificate. This can be retrieve as follows:

```
kubectl -n kube-system \
  get configmap \
  extension-apiserver-authentication \
  -o=jsonpath='{.data.client-ca-file}'
```

### sidecarInjectorImage

`sidecarInjectorImage` is the container image for the sidecar injector.

### secretlessImage

`secretlessImage` is the container image for the Secretless sidecar.

### authenticatorImage

`authenticatorImage` is the container image for the Kubernetes Authenticator sidecar.

### secretsProviderImage

`secretsProviderImage` is the container image for the Secrets Provider sidecar.

### deploymentApiVersion

`deploymentApiVersion` is the supported apiVersion for Deployments. This is the value that
will be set in the Deployment manifest.

`deploymentApiVersion` defaults to the supported apiVersion for Deployments on the latest
Kubernetes release.

If you're on an older version of Kubernetes, you can retrieve the supported apiVersion
for Deployments by running the shell commands below. Update deploymentApiVersion to the
returned.

```bash
deploymentApiGroup=$(kubectl api-resources | \
 grep "deployments.*deploy.*Deployment" | awk '{print $3}')

deploymentApiVersion=$(kubectl api-versions | grep "${deploymentApiGroup}/")

echo ${deploymentApiVersion}
```

#
