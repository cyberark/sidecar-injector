# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Dropped support for Helm V2 and converted to Helm V3.
  [#60](https://github.com/cyberark/sidecar-injector/pull/60)
- K8s APIs used for mutating webhook request/response messages are upgraded
  from the deprecated 'v1beta1' versions to 'v1' so that the Sidecar Injector
  works on Kubernetes v1.22 or newer and OpenShift v4.9 or newer.
  [#62](https://github.com/cyberark/sidecar-injector/pull/62)

### Security
- Added replace statements to go.mod to remove vulnerable dependency versions from the dependency tree
  [cyberark/sidecar-injector#68](https://github.com/cyberark/sidecar-injector/pull/68)
  [cyberark/sidecar-injector#69](https://github.com/cyberark/sidecar-injector/pull/69)

## [0.1.1] - 2020-06-17

### Added
- Add ability to set (and override default) deployment resource `apiVersion` in manifests.
  [#28](https://github.com/cyberark/sidecar-injector/issues/28)

### Changed
- Deployment resource `apiVersion` (in manifests) changed from `extensions/v1beta1` to
  `apps/v1`. [#46](https://github.com/cyberark/sidecar-injector/issues/46)

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

[Unreleased]: https://github.com/cyberark/sidecar-injector/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/cyberark/sidecar-injector/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/cyberark/sidecar-injector/releases/tag/v0.1.0
