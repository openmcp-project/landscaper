---
title: Deploying Resources Created with OCM v2
sidebar_position: 20
---

# Deploying Resources Created with OCM v2

Landscaper can deploy component versions that were built and transferred with the OCM v2 CLI. This page lists what you need, what OCM v2 changes, and how a blueprint deploys an image that OCM v2 stored as a local blob.

## Prerequisites and Limitations

- **Landscaper v1.5.0 or later must already be running on a Kubernetes cluster**, installed as described in [Install the Landscaper Controller](../installation/install-landscaper-controller.md), together with the deployers your blueprints use, such as the Helm deployer. Earlier versions reject OCM v2 component descriptors, because these descriptors have no repository context, and lack the template function [`getImageReference`](./Templating.md).
- **A [`repositoryContext`](./RepositoryContext.md) must already be configured in the [Context](./Context.md#basic-structure) custom resource**, pointing at the registry that holds the component versions. Descriptors created with OCM CLI 0.15 or later carry none.
- **OCM CLI 0.15 or later must already be installed**, as described in [Install the OCM CLI](https://ocm.software/docs/getting-started/install-the-ocm-cli/). It is a new CLI and a new library, and its configuration and its storage layout differ from OCM v1. Read these first:
  - [OCM v2 announcement](https://ocm.software/blog/ocmv2/)
  - [Migrate from Fallback to Deterministic Repository Resolvers](https://ocm.software/docs/how-to/migrate-from-fallback-to-deterministic-repository-resolvers/)
  - [Migrate Legacy Credentials](https://ocm.software/docs/how-to/migrate-legacy-credentials/)

## Deploying an Image From a Local Blob

### Create the Component Version with the Blueprint

#### Write the Blueprint

The blueprint imports the target cluster and points at a deploy execution file. For a Spiff deploy execution, set the `type` to `Spiff`:

```yaml
# blueprint/blueprint.yaml
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Blueprint
jsonSchema: "https://json-schema.org/draft/2019-09/schema"

imports:
  - name: cluster
    type: target
    targetType: landscaper.gardener.cloud/kubernetes-cluster

deployExecutions:
  - name: default
    type: GoTemplate
    file: /deploy-execution.yaml
```

#### Write the Deploy Execution

In the deploy execution, create one deploy item for the Helm deployer. Its image values come from the template function `getImageReference`, which turns the image resource into an OCI reference whether the image is a local blob or not; [Using `getImageReference`](#using-getimagereference) below explains it. As a Go template:

```yaml
# blueprint/deploy-execution.yaml
deployItems:
  - name: podinfo
    type: landscaper.gardener.cloud/helm
    target:
      import: cluster
    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      name: podinfo
      namespace: podinfo
      createNamespace: true
      chart:
        # The chart is a local blob too. resourceRef reads it from the component version.
        resourceRef: {{ getResourceKey `cd://resources/chart` }}
      values:
        {{- $image := getImageReference (getResource .cd "name" "image") }}
        image:
          repository: {{ $image.repository }}
          tag: "{{ $image.tag }}{{ with $image.digest }}@{{ . }}{{ end }}"
```

Or write the same `values` with Spiff, where `&temporary` keeps `ref` out of the rendered values:

```yaml
values:
  ref: (( &temporary( getImageReference(getResource(cd, "name", "image")) ) ))
  image:
    repository: (( ref.repository ))
    tag: (( ref.digest == "" ? ref.tag :ref.tag "@" ref.digest ))
```

#### Write the Constructor

The constructor file holds the blueprint, the Helm chart and the container image. Reference the chart and the image by their original location, here on `ghcr.io`:

```yaml
# component-constructor.yaml
components:
  - name: example.org/my-resource
    version: 1.0.0
    provider:
      name: example.org
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        version: 1.0.0
        input:
          type: dir
          path: ./blueprint
          compress: true
          mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar
      - name: chart
        type: helmChart
        version: 6.14.1
        access:
          type: ociArtifact
          imageReference: ghcr.io/stefanprodan/charts/podinfo:6.14.1
      - name: image
        type: ociImage
        version: 6.14.1
        access:
          type: ociArtifact
          imageReference: ghcr.io/stefanprodan/podinfo:6.14.1
```

#### Create the CTF Archive

Create the component version from the constructor, as a CTF archive:

```shell
ocm add component-version --repository ctf::./ctf --constructor component-constructor.yaml
```

### Transfer the Component Version

#### Transfer with `--copy-resources`

Transfer the CTF into the target OCI registry with `--copy-resources`. This copies the chart and the image into the OCI repository of the component version, so the target registry holds everything the deployment needs:

```shell
ocm transfer component-version ctf::./ctf//example.org/my-resource:1.0.0 \
  registry.example.com/my-component --copy-resources
```

#### Verify the Descriptor

Read the component descriptor back from the target registry:

```shell
ocm get component-version registry.example.com/my-component//example.org/my-resource:1.0.0 -o yaml
```

The image resource now looks like this:

```yaml
component:
  name: example.org/my-resource
  version: 1.0.0
  repositoryContexts: null
  resources:
    - name: image
      type: ociImage
      version: 6.14.1
      relation: external
      access:
        type: LocalBlob/v1
        localReference: sha256:f9537f729129d339aaef049b76ab2cd4ff06a424f99e5f6a5923c10621018fb1
        mediaType: application/vnd.oci.image.index.v1+json
        referenceName: stefanprodan/podinfo:6.14.1
```

Note that the image is still pullable: its manifest sits in the OCI repository of the component version, and `localReference` is its digest. In the target registry, the image is:

```text
registry.example.com/my-component/component-descriptors/example.org/my-resource@sha256:f9537f729129d339aaef049b76ab2cd4ff06a424f99e5f6a5923c10621018fb1
```

Do not rely on `referenceName`. It is only the name the image had before the transfer and does not say where the image is now.

### Using `getImageReference`

The template function `getImageReference` builds the image reference. It reads the registry and path from the repository the component version was actually read from and assembles: the base URL without the scheme, the fixed prefix `component-descriptors`, the component name, the resource version as tag, and `localReference` as digest. With the repository context `https://registry.example.com/my-component`, the image is pullable as:

```text
registry.example.com/my-component/component-descriptors/example.org/my-resource:6.14.1@sha256:f9537f72...
```

The function returns a map with these fields:

- `reference` — the full reference above
- `repository` — the reference without tag and digest
- `tag` — the resource version
- `digest` — the `localReference`

The same call works for an `ociArtifact` access, where it splits the `imageReference`, so the blueprint stays the same whichever access type the image resource has.
The podinfo chart writes the image as `<repository>:<tag>`, so the example appends the digest to `tag`: `6.14.1@sha256:f9537f72...` for a local blob, `6.14.1` for an `ociArtifact` without a digest. Adjust this to how your chart builds the image.
