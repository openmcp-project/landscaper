<img src="./logo/landscaper.svg" width="221">

# Landscaper

[![REUSE status](https://api.reuse.software/badge/github.com/openmcp-project/landscaper)](https://api.reuse.software/info/github.com/openmcp-project/landscaper)
[![Publish](https://github.com/openmcp-project/landscaper/actions/workflows/publish.yaml/badge.svg)](https://github.com/openmcp-project/landscaper/actions/workflows/publish.yaml/badge.svg)
[![Integration Test](https://github.com/openmcp-project/landscaper/actions/workflows/integration_test_main.yaml/badge.svg)](https://github.com/openmcp-project/landscaper/actions/workflows/integration_test_main.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/openmcp-project/landscaper)](https://goreportcard.com/report/github.com/openmp-project/landscaper)

<!-- Motivation -->
_Landscaper_ provides the means to describe, install and maintain cloud-native landscapes. It allows
you to express an order of building blocks, connect output with input data and ultimately, bring your landscape to live.

What does a 'landscape' consist of? In this context it refers not only to application bundles but also includes
infrastructure components in public, private and hybrid environments. 

While tools like Terraform, Helm or native Kubernetes resources work well in their specific problem space, it has been a
manual task to connect them so far. Landscaper solves this specific problem and offers a fully-automated installation
flow. To do so, it translates blueprints of components into actionable items and employs well-known tools like Helm or
Terraform to deploy them. In turn the produced output can be used as input and trigger for a subsequent step -
regardless of the tools used underneath. Since implemented as a set of Kubernetes operators, Landscaper uses the concept
of reconciliation to enforce a desired state, which also allows for updates to be rolled out smoothly.
<!-- end -->

### Start Reading
- The documentation can be found [here](docs/README.md) or you jump directly to the [Guided Tour](docs/guided-tour).
- A list of available deployers is maintained [here](docs/deployer).
- A glossary can be found [here](docs/concepts/Glossary.md)
- Installation instructions can be found [here](docs/installation/install-landscaper-controller.md)

### Information about this fork

This repository is a fork of https://github.com/gardener/landscaper. The support for the Landscaper project is sunsetting in the _Gardener_ organization.
Maintainenance and development of the Landscaper project will continue in the https://github.com/openmcp-project/landscaper repository.
This doesn't affect any feature or functionality of the Landscaper project.
OCI images and OCM components can be consumed directly from within this repository GitHub Container Registry.

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/openmcp-project/service-provider-external-secrets/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure

If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/openmcp-project/service-provider-external-secrets/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/openmcp-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright OpenControlPlane contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/openmcp-project/landscaper).

---

<p align="center">
  <a href="https://apeirora.eu/content/projects/">
    <img alt="BMWK-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="300"/>
  </a>
</p>

<p align="center">
  OpenControlPlane is part of <a href="https://apeirora.eu/content/projects/">ApeiroRA</a>, an EU Important Project of Common European Interest (IPCEI-CIS).
</p>

<p align="center">
  Copyright Linux Foundation Europe. For web site terms of use, trademark policy and other project policies please see <a href="https://linuxfoundation.eu/en/policies">https://linuxfoundation.eu/en/policies</a>.
</p>
