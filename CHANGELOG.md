# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2024-11-15

### Changed
- Upgrade Go to 1.23 (CONJSE-1842, CONJSE-1880)
- Upgrade Alpine to 3.20 and Kubectl to 1.30.3 (CONJSE-1879)

### Added
- Add support for selection of sidecar container versions
  [cyberark/sidecar-injector#71](https://github.com/cyberark/sidecar-injector/pull/71)
- Add ability to set (and override default) deployment resource `apiVersion` in manifests.
    [cyberark/sidecar-injector#27](https://github.com/cyberark/sidecar-injector/pull/27)
- Add support for secrets-provider-for-k8s<br>
  [cyberark/sidecar-injector#72](https://github.com/cyberark/sidecar-injector/pull/72) <br>
  [cyberark/sidecar-injector#73](https://github.com/cyberark/sidecar-injector/pull/73) <br>
  [cyberark/sidecar-injector#74](https://github.com/cyberark/sidecar-injector/pull/74) <br>
  [cyberark/sidecar-injector#79](https://github.com/cyberark/sidecar-injector/pull/79) <br>

### Changed
- Upgrade testify to 1.8.0 and k8s to 0.25.2
  [cyberark/sidecar-injector#77](https://github.com/cyberark/sidecar-injector/pull/78)
- Upgrade Go to 1.19
  [cyberark/sidecar-injector#78](https://github.com/cyberark/sidecar-injector/pull/78)
- Dropped support for Helm V2 and converted to Helm V3.
  [cyberark/sidecar-injector#60](https://github.com/cyberark/sidecar-injector/pull/60)
- K8s APIs used for mutating webhook request/response messages are upgraded
  from the deprecated 'v1beta1' versions to 'v1' so that the Sidecar Injector
  works on Kubernetes v1.22 or newer and OpenShift v4.9 or newer.
  [cyberark/sidecar-injector#62](https://github.com/cyberark/sidecar-injector/pull/62)
- BREAKING CHANGE: Changed annotations to be consistent with other Cyberark repositories.
  sidecar-injector.cyberark.com is changed to conjur.org
  All user manifests must be changed to use the new annotations. [cyberark/sidecar-injector#70](https://github.com/cyberark/sidecar-injector/pull/70)
- Deployment resource `apiVersion` (in manifests) changed from `extensions/v1beta1` to
  `apps/v1`. [#47](https://github.com/cyberark/sidecar-injector/pull/47)
- BREAKING CHANGE: Changed annotation `conjur-token-receivers` to `conjur-inject-volumes` for compatability
  with Secrets Provider [cyberark/sidecar-injector#76](https://github.com/cyberark/sidecar-injector/pull/76)

### Security
- Update alpine base image to 3.18 and golang to 1.21
  [cyberark/sidecar-injector#92](https://github.com/cyberark/sidecar-injector/pull/92)
- Update golang.org/x/net to  v0.17.0
  [cyberark/sidecar-injector#92](https://github.com/cyberark/sidecar-injector/pull/92)
- Upgrade google.golang.org/protobuf to v1.29.1
  [cyberark/sidecar-injector#89](https://github.com/cyberark/sidecar-injector/pull/89)
- Upgraded golang.org/x/net and golang.org/x/text to 0.7.0 to resolve CVE-2022-41723
  [cyberark/sidecar-injector#86](https://github.com/cyberark/sidecar-injector/pull/86)
- Forced golang.org/x/text to use 0.3.8 to resolve CVE-2022-32149
  [cyberark/sidecar-injector#81](https://github.com/cyberark/sidecar-injector/pull/81)
- Added replace statement for golang.org/x/crypto@v0.0.0-20210921155107-089bfa567519
  [cyberark/sidecar-injector#80](https://github.com/cyberark/sidecar-injector/pull/80)
- Updated replace statements to force golang.org/x/net v0.0.0-20220923203811-8be639271d50
   and updated testify to 1.8.0 [cyberark/sidecar-injector#78](https://github.com/cyberark/sidecar-injector/pull/78)
- Added replace statements to go.mod to remove vulnerable dependency versions from the dependency tree
  [cyberark/sidecar-injector#68](https://github.com/cyberark/sidecar-injector/pull/68)
  [cyberark/sidecar-injector#69](https://github.com/cyberark/sidecar-injector/pull/69)
- Updated golang.org/x/net to resolve CVE-2022-41721
  [cyberark/sidecar-injector#84](https://github.com/cyberark/sidecar-injector/pull/84)

## [0.1.0] - 2020-05-28

### Added
- Functional Sidecar Injector with support for the Conjur Kubernetes Authenticator and 
  Secretless Broker. 
- Helm (v2) Chart for deploying Sidecar Injector.
- Ability to inject `conjur-access-token` volume mounts to a selection of containers, in
  tandem with the authenticator sidecar [#25](https://github.com/cyberark/sidecar-injector/issues/25).
   
  The selection of containers are specified via the
  `sidecar-injector.cyberark.com/conjurTokenReceivers` annotation whose value is a
  comma-separated list of container names.
- Ability to configure the sidecar container images by specifying flags on the sidecar
  injector binary [#29](https://github.com/cyberark/sidecar-injector/issues/29).

[Unreleased]: https://github.com/cyberark/sidecar-injector/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/cyberark/sidecar-injector/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cyberark/sidecar-injector/releases/tag/v0.1.0
