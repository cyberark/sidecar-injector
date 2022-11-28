# Solution Design - Generally Available (GA) Conjur Sidecar Injector

## Table of Contents

- [Useful Links](#useful-links)
- [Overview](#overview)
- [Solution](#solution)
- [Design](#design)
- [Performance](#performance)
- [Backwards Compatibility](#backwards-compatibility)
- [Affected Components](#affected-components)
- [Test Plan](#test-plan)
- [Logs](#logs)
- [Documentation](#documentation)
- [Version update](#version-update)
- [Security](#security)
- [Audit](#audit)
- [Development Tasks](#development-tasks)
- [Solution Review](#solution-review)

## Useful Links

| Link | Private |
|------|:-------:|
| [Kubernetes Documentation: MutatingAdmissionWebhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook) | No |
| [Kubernetes Blog: A Guide to Kubernetes Admission Controllers](https://kubernetes.io/blog/2019/03/21/a-guide-to-kubernetes-admission-controllers/) | No |
| [Kubernetes cert-manager Project](https://cert-manager.io/docs/) | No |
| [Kubernetes Documentation: Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/) | No |
| [Sidecar Injector README.md](https://github.com/cyberark/sidecar-injector#cyberark-sidecar-injector) | No |
| [Sidecar Injector Helm chart](https://github.com/cyberark/sidecar-injector/tree/main/charts/cyberark-sidecar-injector) | No |
| [Secrets Provider Documentation](https://docs.cyberark.com/Product-Doc/OnlineHelp/AAM-DAP/11.2/en/Content/Integrations/Kubernetes_deployApplicationsConjur-k8s-Secrets.htm?tocpath=Integrations%7COpenShift%252C%20Kubernetes%252C%20and%20GKE%7CDeploy%20Applications%7C_____3) | No |

## Overview

Sidecars are used in Kubernetes to introduce additional features to application
pods. Manual sidecar injection can be cumbersome and repetitive. The
[Conjur Sidecar Injector](https://github.com/cyberark/sidecar-injector)
enables automatic sidecar injection as an instantiation of a
[Kubernetes Mutating Admission Webhook Controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook).
This means when CyberArk Sidecar Injector is deployed and enabled in a
Namespace, any Pod created in that namespace with the appropriate Annotations
will result in automated sidecar injection.

### Admission controllers:
An admission controller intercepts Kubernetes API requests before the objects
are persisted. There are two special types of controllers, the MutatingAdmissionWebhook
and ValidatingAdmissionWebhook.
Mutating webhooks are called in the mutating phase and may modify the object.
For the side-car injector, the modification applied is to add a sidecar.

![Admission Controller Phases](./admission-controller-phases.png)

The Conjur Sidecar Injector provides support for a selection of available
sidecars that are configurable through annotations.

The Conjur Sidecar Injector is currently in beta. This document describes
the efforts necessary to bring the Conjur Sidecar Injector to the "General availability" (Version 1.0) status.

### User Experience Improvements

- Simplification of Application Namespace Prep
  - Sidecar Injector can inject Conjur connection info directly into injected
    authenticator containers as environment variables, rather than requiring
    the application deployer to create Conjur connect ConfigMap in the
    application Namespace.
  - If Sidecar injector can also generate a RoleBinding when it mutates a
    Pod definition, then the deployer will no longer need to do a Helm install
    of the Namespace Prep Helm chart.
- Support for Additional CyberArk Authenticator Containers
  - Secrets Provider as sidecar with authn-k8s
  - Secrets Provider as sidecar with authn-jwt
  - Secrets Provider in standalone mode??? (User deploys empty Job with
    Conjur sidecar injection Annotations, Sidecar Injector adds Secrets
    Provider container)
- Support for Selection of Sidecar Container Versions/Tags via Annotations
  Currently, versions are hardcoded to `latest` for each injected container image.
- Sidecar Injector Annotations Made Consistent with Secrets Provider Annotations
- OpenTelemetry Tracing support??? (helpful for latency measurements???)
- INVESTIGATION SPIKE REQUIRED: Use Kubernetes cert-manager to Simplify Signing of CSR<br />
  - Currently, the Sidecar Injector installation process requires a human (e.g.
  a Kubernetes admin) to approve a Kubernetes Certificate Signing Request
  (CSR).
  - In some customer environments, it may be possible to use the
  [Kubernetes cert-manager project](https://cert-manager.io/docs/)
  to simplify/automate the approval/signing process.

### Quality and Development Environment Improvements

- UT coverage improvements (currently at 68.8% coverage)
- Create special Sidecar Injector image that includes Go test coverage
  instrumentation (as was done for Secretless Broker)
- Merge integration test coverage with UT test coverage
- Automated Testing improvements:

  Current CI/local integration testing includes:
  - Secretless Broker sidecar, but only using Custom Resource Definition (CRD) config
  - Sidecar Injector deployed via Helm chart only
  - GKE testing only for Jenkins CI, or KinD testing for local runs
  - NO Conjur integration testing (uses hardcoded username/password in
    Secretless config, rather than retrieving Conjur variables)
  - NO testing of authn-k8s client container <br/>

  New CI testing will be added:
  - Testing against a Conjur OSS instance
  - Testing against a Conjur Enterprise instance and decomposed follower.
  - Testing of Secrets Provider using authn-k8s, Push to File
  - Testing of Secrets Provider using authn-k8s, Kubernetes Secrets mode
  - Testing of Secrets Provider using authn-jwt, Push to File
  - Testing of Secrets Provider using authn-jwt, Kubernetes Secrets mode
  - Testing of authn-k8s client container
  - Testing of Secretless Broker container using secrets.yml config
  - Testing in OpenShift (4.8 or newer)
  - Testing of Secrets Provider installation using raw Kubernetes manifests
- New Helm Chart Unit/Schema testing
  - Add Helm chart schema files
  - Add Helm chart Unit tests based on [unittest plugin](https://github.com/quintush/helm-unittest))
  - Add Helm schema tests based on Helm lint
- Add Helm chart test to verify that an installed Sidecar Injector can inject containers that can properly authenticate with Conjur
  - Similar to Helm test implemented for Cluster Prep Helm chart
  - Deploys a simple, BATS-based test app that gets injected with an authenticator sidecar
  - After injection, the test app attempts to authenticate with Conjur using special low-privilege
    Conjur host
 
### Project Scope

- Supported/Tested with Conjur OSS and Conjur Enterprise
- Supported/Tested in GKE and OpenShift
- Requires Kubernetes 1.21 or newer, or OC 4.8 or newer

### Investigation Spikes:

- Does the Sidecar Injector already support filtering of Namespaces for
  which it's monitoring for Pods to mutate based on labels applied to the
  Namespaces? I vaguely remember this filtering being in place. If not,
  then this feature should be included in this design for scaling reasons
  and for added security (only inject in Namespaces labeled for injection).
  
- The Secrets Provider supports synchronization of container startup
  using a `postStart` container lifecycle hook so that application
  container(s) are not started up until after the Secrets Provider has
  completed its first round of providing secrets. However,
  **this feature requires that the Secrets Provider be listed FIRST
  in the list of containers in a Pod**. Patching of a Pod likely places
  new, injected containers at the end of the current set of containers.
  How can we force the ordering of containers such that an injected
  Secrets Provider container is listed first in the Pod spec?

- Can Injector also create a RoleBinding when it mutates a Pod? If so, we
  can eliminate the Namespace Prep Helm chart install. According to Kubernetes
  documentation:<br /><br />
  _Mutating controllers may modify related objects to the requests they admit;
  validating controllers may not._<br /><br />
  (Reference: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#what-are-they)<br />

- Can we use the Kubernetes cert-manager project to simplify/automate the
  signing of the SI's CSR?

### Wish List:

- Can the Follower Operator also deploy the sidecar injector?
- **Secretless Broker work**: Use Labels instead of CRD Name Suffixes<br />
  Currently, in order to support multiple instances of the Secretless Broker
  when configuration is done via CRDs, each Secretless Broker instance must
  be configured to watch for Custom Resources (CRs) for CRDs with unique names.
  This is achieved by using a unique CRD suffix for CRD names for each
  instance of Secretless Broker. It would be much simpler, and more
  idiomatic, to use distinct labels applied to CRs, and have the Secretless
  Broker instances filter based on these labels.

- Removing support of injection of _**init containers**_:
Secrets Provider supports synchronized startup of containers when running
as a sidecar container. The authenticator container can also be enhanced
to support a sidecar mode with synchronized startup support using a
`postStart` lifecycle hook. If we implement this enhancement, then init
containers can theoretically be deprecated.

- Check if the manifest already has the same sidecar configured as it being injected.
Injecting a duplicate sidecar may result in errors.

## Solution

### User Experience

#### Feature Details


#### No Longer Supported Pod Annotations

| Annotations | Description | Why No Longer Supported |
|-------------|-------------|-------------------------|
| sidecar-injector.cyberark.com/conjurAuthConfig | ConfigMap holding Conjur authentication configuration | CONJUR_AUTHN_LOGIN set directly via conjur.org/conjur-authn-login |
| sidecar-injector.cyberark.com/conjurConnConfig | ConfigMap holding Conjur connection configuration | Env vars injected directly based on content of Sidecar Injector's local Conjur ConfigMap) |

#### New, Renamed, or Existing Pod Annotations for Configuring Sidecar Injection

_**QUESTIONS: How should we make new Annotation keys consistent with Secrets
 Provider Annotation keys? Should keys be a flat hierarchy under `conjur.org`?**_

| Annotation | Already Exists? | Replaces Annotation | Description, Notes | Default |
|------------|-----------------|---------------------|--------------------|---------|
| conjur.org/inject | No | sidecar-injector.cyberark.com/inject | Set to true to enable sidecar injection | false |
| conjur.org/secretless-config | No | sidecar-injector.cyberark.com/secretlessConfig | ConfigMap holding Secretless configuration | (required for secretless) |
| conjur.org/inject-type | No | sidecar-injector.cyberark.com/injectType | Injected Sidecar type (secretless, authenticator, or secrets-provider) | (required) |
| conjur.org/conjur-token-receivers | No | sidecar-injector.conjurTokenReceivers | Comma-separated list of the names of containers, in the pod, that will be injected with conjur-access-token VolumeMounts. (e.g. app-container-1,app-container-2) | (only required for authenticator) |
| conjur.org/container-mode | Yes | sidecar-injector.cyberark.com/containerMode | Sidecar Container mode (init or sidecar) | (required for authenticator or secrets-provider) |
| conjur.org/container-name | No | sidecar-injector.cyberark.com/containerName | Sidecar Container name | (required only for authenticator) |
| conjur.org/conjur-authn-login | No | (NEW) | Sets the CONJUR_AUTHN_LOGIN for authenticator | (required for authenticator) |
| conjur.org/container-image | No | (NEW) | Sets the container image to inject | Defaults to "latest" version of image appropriate for the container type |
| conjur.org/container-image-pull-policy | No | (NEW) | Sets the image pull policy for the injected container | "Always" |
| conjur.org/restart-app-via-liveness | No | (NEW) | Allows configuration of a livenessProbe on the app container that restarts that container when secrets have changed | false (only applies to Secrets Provider) |


#### "Pass-Through" Annotations (Ignored by Sidecar Injector)

All existing Pod Annotations that are defined for the Secrets Provider, except
for `conjur.org/container-mode`, are ignored (and left intact) by the Sidecar
Injector. These Annotations are expected to be parsed by the Secrets Provider
container that is getting injected. The `conjur.org/container-mode`
Annotation is expected to be parsed by both the Sidecar Injector and by any
injected Secrets Provider container.

### Project Scope and Limitations

The initial implementation and testing will be limited to:

- Sidecar Container configurations to be supported/tested:

  - Secretless Broker sidecar container, configured via secretless.yml in a ConfigMap
  - Authenticator sidecar container
  - Secrets Provider sidecar/init container, authn-k8s, Push-to-File mode
  - Secrets Provider sidecar/init container, authn-k8s, Kubernetes Secrets mode
  - Secrets Provider sidecar/init container, authn-jwt, Push-to-File mode
  - Secrets Provider sidecar/init container, authn-jwt, Kubernetes Secrets mode

- Platforms:

  - Kubernetes 1.21 or newer
  - OpenShift versions old, current and newest as defined by infrastructure
    (currently 4.6, 48 and 4.9 at the time of this writing)

- Scale:

  Scaling limits defined for the Secrets Provider:

  - Support for up to 50 DAP secrets per Secrets Provider init container,
    where the variable paths are, on average, 100 characters.

#### Out of Scope

OCP versions not supported by infrastructure.

##  Design

### Eliminating/Simplifying Application Namespace Preparation
The Conjur Namespace prep Helm chart reads the golden config and creates a local
configmap. This action can also be done in the sidecar injector.

## Performance

If OpenTelemetry Tracing is added to the Sidecar Injector, then this tracing
feature can be used to measure the incremental delay/latency for
injecting containers into Pods.

## Backwards Compatibility

This design represents a major breaking change for the Sidecar Injector
because of the changes in Annotations supported.

## Affected Components

This feature affects the Sidecar Injector component. No other components
will be affected.

## Test Plan

### Test environments

### Test assumptions

### Out of scope

### Test prerequisites

### Test Cases

#### Unit Tests

#### E2E tests

E2E tests are expensive. We should limit E2E testing to happy path "smoke
tests", and leave sad path testing to unit testing and lower level functional
tests.

The current E2E tests deploy a test pod with an Echo Server. If a full deployment with the
Pet Store app in needed for E2E tests, instead of duplicating tests, the sidecar injector can be
added to the conjur-authn-k8s-client tests and then sidecar-injector could clone the 
repo and run the specific tests with side-car injector.

#### Security testing

Security testing will include:

- Automated Vulnerability scans

####  Performance testing

## Logs

## Documentation

## Version update

## Security

The Sidecar Injector must be capable of reading and modifying all Pods
in all Namespaces that are labeled for Conjur Sidecar Injection.
This is not expected to be a security concern, since Pod definitions are
not sensitive information, and this access to reading/modifying Pods should
be expected for any Sidecar injector.

The Sidecar Injector must not mutate any Pods that are in Namespaces
that are not labeled for Conjur Sidecar Injection. The 
[ namespaceSelector](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-namespaceselector)
can be used to limit the pods information that is sent to the sidecar injector to specific
labelled namespaces.

Questions:
- Should we turn off ssh to prevent an attacker from logging into the 
sidecar-injector container except for debug mode?
- Do we need sanity checks for the pod configuration sent to the sidecar-injector?
- Can the tracing 


## Audit

No changes to Conjur audit behavior are required for this feature.

##  Development Tasks

Development tasks for this feature are organized into tasks corresponding to
three phases of development:

- Minimally-featured community release
- Full-featured GA release

### Development Tasks: Minimally-Featured Community Release

| Description | Jira Story | Estimated<br />Story<br />Points<br /> | Completed |
|-------------|------------|-----------|-----------|
| Annotations Made Consistent with Secrets Provider Annotations |ONYX-19422|||
| Simplification of Application Namespace Prep |ONYX-19423|||
| Support for Secrets Provider Container |ONYX-19424|||
| Support for Selection of Sidecar Container Versions |ONYX-19425|||
| Simplification of Application Namespace Prep - rolebinding |ONYX-19489|||
| Create documentation for tech writers |ONYX-20482|||
| Add e2e testing via conjur-authn-k8s-client|ONYX-21139|||
### Development Tasks: Full-Featured GA Release

| Description | Jira Story | Estimated<br />Story<br />Points<br /> | Completed |
|-------------|------------|-----------|-----------|
| OpenTelemetry Tracing support |ONYX-19426|||
| UT coverage improvements |ONYX-19427|||
| Sidecar Injector image with Go test coverage |ONYX-19435|||
| New CI testing |ONYX-19436|||
| [INVESTIGATION SPIKE] Use Kubernetes cert-manager to Simplify Signing of CSR |ONYX-19428|||
| Helm Chart Unit/Schema testing |ONYX-19437|||
| Helm chart test |ONYX-19438|||
| Add Test Level/Area Tags to Support Improve Dev Feedback - Sidecar-injector |ONYX-19485|||
| Manual UX check on public docs|ONYX-20481|||

### Development Tasks: Future

| Description | Jira Story | Estimated<br />Story<br />Points<br /> | Completed |
|-------------|------------|-----------|-----------|

## Solution Review

<table>
<thead>
<tr class="header">
<th><strong>Persona</strong></th>
<th><strong>Name</strong></th>
<th><strong>Design Approval</strong></th>
</tr>
</thead>
<tbody>
<tr class="odd">
<td>Team Leader</td>
<td></td>
<td><ul>
<p> </p>
</ul></td>
</tr>
<tr class="even">
<td>Product Owner</td>
<td>Jane Simon</td>
<td><ul>
<p> </p>
</ul></td>
</tr>
<tr class="odd">
<td>System Architect</td>
<td>Rafi Schwarz</td>
<td><ul>
<p> </p>
</ul></td>
</tr>
<tr class="even">
<td>Security Architect</td>
<td>Andy Tinkham</td>
<td><ul>
<p> </p>
</ul></td>
</tr>
<tr class="odd">
<td>QA Architect</td>
<td>Adam Ouamani</td>
<td><ul>
<p> </p>
</ul></td>
</tr>
</tbody>
</table>
