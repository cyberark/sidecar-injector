# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Ability to inject `conjur-access-token` volume mounts to a selection of containers, in
  tandem with the authenticator sidecar [#25](https://github.com/cyberark/sidecar-injector/issues/25).
   
  The selection of containers are specified via the
  `sidecar-injector.cyberark.com/conjurTokenReceivers` annotation whose value is a
  comma-separated list of container names.
- Ability to configure the sidecar container images by specifying flags on the sidecar
  injector binary [#29](https://github.com/cyberark/sidecar-injector/issues/29).
