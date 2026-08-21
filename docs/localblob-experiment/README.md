# localBlob delivery experiment

Delivers a self-contained OCM component version through an intermediate registry and
deploys it with a Landscaper built from this workspace.

```console
$ ./run-experiment.sh all
```

This is a scratch experiment, not part of the [guided tour](../guided-tour). It shares no
code, no cluster and no registry with it, and it creates its own kind cluster.

## What it shows

After `ocm transfer --copy-resources`, a component version is a
single OCI index carrying the descriptor, the blueprint, the helm chart and the container
image. Nothing in it points back at the registry it was built from, so a plain `oras cp` —
no OCM tooling on that leg — delivers the whole thing, and the pod ends up pulling its
image from the component's own repository, by digest.

```
component-constructor.yaml   podinfo chart + image, by reference to ghcr.io
  │ ocm add component-version
  ▼ CTF
  │ ocm transfer --copy-resources
  ▼ vendor registry       http://localhost:5002/vendor        <- stands in for a delivery
  │ oras cp                                                      (no OCM tooling here)
  ▼ in-cluster registry   http://localhost:5555/internal      <- via port-forward
  │ Landscaper
  ▼ pod pulls oci-registry...:5000/internal/component-descriptors/...@sha256:...
```

The vendor registry is a throwaway docker container, so the delivery leg needs no
credentials and no network egress. Only the initial build reaches ghcr.io, for podinfo.

The step that makes this work is in the blueprint. A local blob has no
`access.imageReference`, so the image reference is assembled from the repository context
of the `Context` custom resource, the component name, and the `localReference` digest:

```yaml
{{ $imageResource := getResource .cd "name" "image" }}
{{ $repoBase := trimPrefix "https://" (trimPrefix "http://" .componentDescriptorDef.ref.repositoryContext.baseUrl) }}
image:
  repository: {{ $repoBase }}/component-descriptors/{{ .cd.component.name }}
  tag: "{{ .cd.component.version }}@{{ $imageResource.access.localReference }}"
```

## Prerequisites

- OCM **v2** CLI at `~/.local/bin/ocm` — `./run-experiment.sh install-ocm` fetches it
- `docker`, `kind`, `kubectl`, `helm`, `oras`, `curl`, `jq`, `yq` (v4, mikefarah)
- network access to ghcr.io for the podinfo chart and image

## Cases

Run `all` for the whole thing, or any single case on its own.

### The experiment

| Case | What it does |
| --- | --- |
| `setup` | Write the blueprint, the constructor and the k8s manifests |
| `component` | 1. build the CTF |
| `transfer` | 2. `--copy-resources` into the vendor registry |
| `copy` | 3. `oras cp` from the vendor into the in-cluster registry |
| `verify` | The image manifest is pullable by digest from the component's repository |
| `deploy` | 4. Context, Target, Installation; wait, then show what got deployed |
| `descriptor` | Print the component descriptor as read from the in-cluster registry |
| `status` | Installation, DeployItem, rendered values, pods |
| `clean` | Remove the k8s resources and every generated file; keep the cluster |

### The environment

| Case | What it does |
| --- | --- |
| `up` | `cluster` + `images` + `install` + `vendor-registry`, from nothing |
| `cluster` | Create the kind cluster if it is not there yet |
| `install-ocm` | Install the OCM v2 CLI into the directory of `$OCM` |
| `images` | Build the workspace binaries and images, load them into kind |
| `install` | `helm upgrade --install` landscaper + helm/manifest deployer with those images |
| `reinstall` | Uninstall the three releases first, then install |
| `registry` | Install the in-cluster OCI registry (called by `install`) |
| `node-config` | Let the kind node pull from that registry (called by `registry`) |
| `vendor-registry` | Start the host-side docker registry that stands in for a delivery |
| `down` | Delete the kind cluster **and** the vendor registry container |

## Generated files

The script lives in the tracked docs tree, so everything it generates — blueprint,
constructor, CTF, manifests, kubeconfig, logs — goes to `tmp/localblob-experiment/`
instead. Override with `STATE_DIR`.

## Environment overrides

`OCM`, `KIND_CLUSTER`, `NAMESPACE`, `DEPLOY_NAMESPACE`, `LOCAL_PORT`, `VENDOR_PORT`,
`IMAGE_TAG`, `STATE_DIR`, `NODE_IMAGE`, `REGISTRY_IMAGE`, `REGISTRY_SIZE`, `WAIT_TIMEOUT`.

To run on an existing guided-tour cluster instead of creating a second one:

```console
$ KIND_CLUSTER=ls ./run-experiment.sh all
```

Two notes on ports. Host port 5000 is taken by the macOS AirPlay receiver, which is why the
port-forward uses 5555. And the v2 CLI has no `--insecure` flag and does not special-case
localhost — an explicit `http://` scheme in the reference is the only way to make it speak
plain HTTP, so every repository reference here carries one.
