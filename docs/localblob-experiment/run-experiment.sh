#!/bin/bash
# Delivers a self-contained OCM component version through an intermediate registry and
# deploys it with a Landscaper built from this workspace.
#
# The point of the experiment is the middle leg. After
#   ocm transfer --copy-resources
# the component version is a single OCI index carrying the descriptor, the blueprint, the
# helm chart and the container image. A plain "oras cp" -- no OCM tooling on that leg --
# therefore moves all of it, and nothing in the delivered component points back at the
# registry it was built from. The pod ends up pulling its image from the component's own
# repository, by digest.
#
#   component-constructor.yaml   podinfo chart + image, by reference to ghcr.io
#     | ocm add component-version
#     v CTF
#     | ocm transfer --copy-resources
#     v vendor registry     http://localhost:5002/vendor      <- stands in for a delivery
#     | oras cp             (no OCM tooling on this leg)
#     v in-cluster registry http://localhost:5555/internal    <- via port-forward
#     | Landscaper
#     v pod pulls oci-registry...:5000/internal/component-descriptors/...@sha256:...
#
# Everything is local: the vendor registry is a docker container, so the delivery needs no
# credentials and no network egress. Only the initial build reaches ghcr.io for podinfo.
#
# This script is deliberately standalone -- it shares no code, no cluster and no registry
# with docs/guided-tour. Set KIND_CLUSTER=ls to run it on an existing guided-tour cluster
# instead of creating its own.
#
# Prerequisites:
#   - OCM v2 CLI at ~/.local/bin/ocm (override with OCM=...); "install-ocm" fetches it
#   - docker, kind, kubectl, helm, oras, curl, jq, yq
#   - network access to ghcr.io for the podinfo chart and image
#
# Usage:
#   ./run-experiment.sh all          # everything, from nothing
#   ./run-experiment.sh <case>
#
# Cases -- the experiment:
#   setup        Write the blueprint, the constructor and the k8s manifests
#   component    1. build the CTF
#   transfer     2. --copy-resources into the vendor registry
#   copy         3. oras cp from the vendor into the in-cluster registry
#   verify       The image manifest is pullable by digest from the component's repository
#   deploy       4. Context, Target, Installation; wait, then show what got deployed
#   descriptor   Print the component descriptor as read from the in-cluster registry
#   status       Installation, DeployItem, rendered values, pods
#   clean        Remove the k8s resources and every generated file; keep the cluster
#
# Cases -- the environment:
#   up           cluster + images + install + vendor-registry, from nothing
#   cluster      Create the kind cluster if it is not there yet
#   install-ocm  Install the OCM v2 CLI into the directory of $OCM
#   images       Build the branch binaries and images, load them into kind
#   install      helm upgrade --install landscaper + helm/manifest deployer with those images
#   reinstall    Uninstall the three releases first, then install
#   registry     Install the in-cluster OCI registry (called by install)
#   node-config  Let the kind node pull from that registry (called by registry)
#   vendor-registry  Start the host-side docker registry that stands in for a delivery
#   down         Delete the kind cluster AND the vendor registry container
#
# Environment overrides:
#   OCM, KIND_CLUSTER, NAMESPACE, DEPLOY_NAMESPACE, LOCAL_PORT, VENDOR_PORT, IMAGE_TAG,
#   STATE_DIR, NODE_IMAGE, REGISTRY_IMAGE, REGISTRY_SIZE, WAIT_TIMEOUT

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$SCRIPT_DIR"

# The script lives in the tracked docs tree, so every generated file -- blueprint,
# constructor, CTF, manifests, kubeconfig, logs -- goes to a scratch directory under the
# gitignored tmp/ instead.
STATE_DIR="${STATE_DIR:-$REPO_ROOT/tmp/localblob-experiment}"
OUT="$STATE_DIR"

OCM="${OCM:-$HOME/.local/bin/ocm}"

KIND_CLUSTER="${KIND_CLUSTER:-localblob-exp}"
NODE_IMAGE="${NODE_IMAGE:-kindest/node:v1.34.0}"
KUBECTX="kind-$KIND_CLUSTER"
KUBECTL=(kubectl --context "$KUBECTX")

# Namespace holding the Context, Target and Installation, and the one the chart deploys to.
NAMESPACE="${NAMESPACE:-localblob-exp}"
DEPLOY_NAMESPACE="${DEPLOY_NAMESPACE:-localblob-exp-example}"

COMPONENT="example.org/localblob-demo"
VERSION="1.0.0"

# The in-cluster registry is reached under two addresses: a port-forward from the host for
# oras, and the cluster-internal service name for the Landscaper's repositoryContext.
# Port 5000 on the host is taken by the macOS AirPlay receiver, hence 5555.
LOCAL_PORT="${LOCAL_PORT:-5555}"
REGISTRY_NS="landscaper"
REGISTRY_SVC="oci-registry"
REGISTRY_IMAGE="${REGISTRY_IMAGE:-registry:2}"
REGISTRY_SIZE="${REGISTRY_SIZE:-5Gi}"
REGISTRY_CLUSTER_HOST="${REGISTRY_SVC}.${REGISTRY_NS}.svc.cluster.local:5000"

# The vendor registry stands in for the registry a component is delivered through.
VENDOR_PORT="${VENDOR_PORT:-5002}"
VENDOR_CONTAINER="${KIND_CLUSTER}-vendor"

# The v2 CLI has no --insecure flag and does not special-case localhost; an explicit http://
# scheme in the reference is the only way to make it speak plain HTTP.
REPO_VENDOR="http://localhost:${VENDOR_PORT}/vendor"
REPO_HOST="http://localhost:${LOCAL_PORT}/internal"
REPO_CLUSTER="http://${REGISTRY_CLUSTER_HOST}/internal"

# Paths below the registry root, for the raw HTTP checks that oras and curl need.
VENDOR_PATH="vendor/component-descriptors/${COMPONENT}"
COMPONENT_PATH="internal/component-descriptors/${COMPONENT}"

IMAGE_TAG="${IMAGE_TAG:-dev-$(git -C "$REPO_ROOT" rev-parse --short HEAD)}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-600}"
IMAGES="landscaper-controller landscaper-webhooks-server helm-deployer-controller manifest-deployer-controller"

mkdir -p "$OUT"

# run echoes each command before executing it. The banner goes to stderr so that
# digest=$(run ...) captures the command's output and not the banner.
run() { echo "+ $*" >&2; "$@"; }

green=$'\033[32m'
reset=$'\033[0m'
header() { printf '\n%s=== %s ===%s\n' "$green" "$*" "$reset"; }

# ensure_ocm installs the CLI when the configured path does not hold one yet.
ensure_ocm() {
  [ -x "$OCM" ] && return 0
  echo "no OCM CLI at ${OCM}"
  "$0" install-ocm
}

port_forward_pid=""

# require_port_forward opens the registry port-forward and closes it when the script exits.
require_port_forward() {
  if curl -s -o /dev/null -m 1 "http://localhost:${LOCAL_PORT}/v2/"; then
    return 0
  fi
  echo "+ kubectl port-forward -n ${REGISTRY_NS} svc/${REGISTRY_SVC} ${LOCAL_PORT}:5000" >&2
  "${KUBECTL[@]}" port-forward -n "$REGISTRY_NS" "svc/${REGISTRY_SVC}" "${LOCAL_PORT}:5000" \
    >"$OUT/port-forward.log" 2>&1 &
  port_forward_pid=$!
  trap 'kill "$port_forward_pid" 2>/dev/null || true' EXIT
  for _ in $(seq 1 30); do
    curl -s -o /dev/null -m 1 "http://localhost:${LOCAL_PORT}/v2/" && return 0
    sleep 1
  done
  echo "registry not reachable on localhost:${LOCAL_PORT}, see ${OUT}/port-forward.log" >&2
  exit 1
}

# http_code prints just the status code, for checks where a 404 is the finding.
http_code() {
  curl -s -o /dev/null -w "%{http_code}" \
    -H "Accept: application/vnd.oci.image.index.v1+json" \
    -H "Accept: application/vnd.oci.image.manifest.v1+json" \
    "http://localhost:${LOCAL_PORT}/v2/$1/manifests/$2"
}

# image_local_reference reads the digest the image resource was stored under. It selects the
# resource by name rather than by position, so a reordered descriptor cannot yield the
# digest of a different resource.
image_local_reference() {
  "$OCM" get component-version "${REPO_HOST}//${COMPONENT}:${VERSION}" -o json 2>/dev/null |
    jq -r '.[0].component.resources[] | select(.name == "image") | .access.localReference'
}

# settle waits until the installation's current job finished, not merely for a terminal
# phase -- re-applying the same Installation would otherwise show the previous run's
# Succeeded before the new job has even started.
settle() {
  local name="$1" deadline=$((SECONDS + WAIT_TIMEOUT)) job fin phase=""
  echo "+ waiting up to ${WAIT_TIMEOUT}s for installation ${name} to settle" >&2
  while [ "$SECONDS" -lt "$deadline" ]; do
    job=$("${KUBECTL[@]}" -n "$NAMESPACE" get installation "$name" \
      -o jsonpath='{.status.jobID}' 2>/dev/null || true)
    fin=$("${KUBECTL[@]}" -n "$NAMESPACE" get installation "$name" \
      -o jsonpath='{.status.jobIDFinished}' 2>/dev/null || true)
    phase=$("${KUBECTL[@]}" -n "$NAMESPACE" get installation "$name" \
      -o jsonpath='{.status.phase}' 2>/dev/null || true)
    if [ -n "$job" ] && [ "$job" = "$fin" ]; then
      case "$phase" in
        Succeeded) echo "  ${name}: Succeeded"; return 0 ;;
        Failed|DeleteFailed)
          echo "  ${name}: ${phase}" >&2
          "${KUBECTL[@]}" -n "$NAMESPACE" get installation "$name" -o yaml |
            yq '.status' >&2
          return 1 ;;
      esac
    fi
    sleep 5
  done
  echo "  ${name}: timed out in phase '${phase:-<none>}'" >&2
  return 1
}

# apply_target points a Target at this same cluster. --internal yields the api server
# address reachable from inside the cluster; the plain kubeconfig says 127.0.0.1, which
# inside a pod is the pod.
apply_target() {
  "${KUBECTL[@]}" create namespace "$NAMESPACE" --dry-run=client -o yaml |
    "${KUBECTL[@]}" apply -f -
  kind get kubeconfig --name "$KIND_CLUSTER" --internal >"$OUT/kubeconfig-internal.yaml"
  {
    cat <<EOF
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Target
metadata:
  name: my-cluster
  namespace: $NAMESPACE
spec:
  type: landscaper.gardener.cloud/kubernetes-cluster
  config:
    kubeconfig: |
EOF
    sed 's/^/      /' "$OUT/kubeconfig-internal.yaml"
  } | "${KUBECTL[@]}" apply -f -
}

show_status() {
  header "Installation and DeployItem in ${NAMESPACE}"
  run "${KUBECTL[@]}" get installation,deployitem -n "$NAMESPACE" -o wide || true
  for di in $("${KUBECTL[@]}" get deployitem -n "$NAMESPACE" -o name 2>/dev/null); do
    echo "--- $di rendered values"
    "${KUBECTL[@]}" get "$di" -n "$NAMESPACE" -o jsonpath='{.spec.config.values}' | jq . || true
    echo
    echo "--- $di last error"
    "${KUBECTL[@]}" get "$di" -n "$NAMESPACE" -o jsonpath='{.status.lastError.message}' || true
    echo
  done
  header "Deployment and pods in ${DEPLOY_NAMESPACE} -- note where the image is pulled from"
  run "${KUBECTL[@]}" get deployment,pods -n "$DEPLOY_NAMESPACE" -o wide || true
}

case "${1:-}" in

  ##### the experiment #####

  setup)
    header "Setup: blueprint, constructor and k8s manifests in ${OUT}"
    rm -rf "$OUT/blueprint" "$OUT/manifests"
    mkdir -p "$OUT/blueprint" "$OUT/manifests"

    cat >"$OUT/blueprint/blueprint.yaml" <<EOF
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
EOF

    # The Go template variables and the backticks are escaped so the shell leaves them to
    # the Landscaper; only DEPLOY_NAMESPACE is substituted here.
    cat >"$OUT/blueprint/deploy-execution.yaml" <<EOF
deployItems:
  - name: default-deploy-item
    type: landscaper.gardener.cloud/helm

    target:
      import: cluster

    config:
      apiVersion: helm.deployer.landscaper.gardener.cloud/v1alpha1
      kind: ProviderConfiguration
      name: podinfo
      namespace: ${DEPLOY_NAMESPACE}
      createNamespace: true

      chart:
        resourceRef: {{ getResourceKey \`cd://resources/chart\` }}

      values:
        # A resource stored as a local blob has no access.imageReference. The image
        # reference is assembled from the repository context (taken from the Context
        # custom resource), the component name, and the digest in the localReference.
        {{ \$imageResource := getResource .cd "name" "image" }}
        {{ \$repoBase := trimPrefix "https://" (trimPrefix "http://" .componentDescriptorDef.ref.repositoryContext.baseUrl) }}
        image:
          repository: {{ \$repoBase }}/component-descriptors/{{ .cd.component.name }}
          tag: "{{ .cd.component.version }}@{{ \$imageResource.access.localReference }}"
EOF

    # Media type +tar, not +tar+gzip: the v2 CLI appends +gzip itself when compress is set.
    cat >"$OUT/component-constructor.yaml" <<EOF
components:
  - name: ${COMPONENT}
    version: ${VERSION}
    provider:
      name: internal
    resources:
      - name: blueprint
        type: landscaper.gardener.cloud/blueprint
        version: ${VERSION}
        input:
          type: dir
          path: ./blueprint
          compress: true
          mediaType: application/vnd.gardener.landscaper.blueprint.v1+tar
      - name: chart
        type: helmChart
        version: "6.14.1"
        access:
          type: ociArtifact
          imageReference: ghcr.io/stefanprodan/charts/podinfo:6.14.1
      - name: image
        type: ociImage
        version: "6.14.1"
        access:
          type: ociArtifact
          imageReference: ghcr.io/stefanprodan/podinfo:6.14.1
EOF

    cat >"$OUT/manifests/context.yaml" <<EOF
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Context
metadata:
  name: localblob-demo
  namespace: ${NAMESPACE}
repositoryContext:
  baseUrl: ${REPO_CLUSTER}
  type: ociRegistry
EOF

    cat >"$OUT/manifests/installation.yaml" <<EOF
apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: localblob-demo
  namespace: ${NAMESPACE}
  annotations:
    landscaper.gardener.cloud/operation: reconcile
spec:
  context: localblob-demo
  componentDescriptor:
    ref:
      componentName: ${COMPONENT}
      version: ${VERSION}
  blueprint:
    ref:
      resourceName: blueprint
  imports:
    targets:
      - name: cluster
        target: my-cluster
EOF
    echo "wrote ${OUT}/{blueprint,component-constructor.yaml,manifests}"
    ;;

  component)
    ensure_ocm
    header "1. Build the CTF -- chart and image are still references to ghcr.io"
    rm -rf "$OUT/ctf"
    # cd into OUT: the v2 CLI resolves the constructor's input paths from the CWD.
    ( cd "$OUT" && run "$OCM" add component-version \
        --repository "ctf::$OUT/ctf" \
        --constructor component-constructor.yaml )
    ;;

  transfer)
    ensure_ocm
    "$0" vendor-registry
    header "2. Transfer into the vendor registry, resources as local blobs"
    run "$OCM" transfer component-version \
      "ctf::$OUT/ctf//${COMPONENT}:${VERSION}" "$REPO_VENDOR" \
      --copy-resources

    header "2. The descriptor in the vendor registry -- accesses are digests, not addresses"
    run "$OCM" get component-version "${REPO_VENDOR}//${COMPONENT}:${VERSION}" -o json |
      jq '.[0]'
    ;;

  copy)
    require_port_forward
    header "3. oras cp from the vendor into the in-cluster registry -- no OCM tooling"
    # One tag carries the descriptor, the blueprint, the chart and the image, so a plain
    # registry-to-registry copy is enough to deliver the whole component.
    run oras cp --from-plain-http --to-plain-http \
      "localhost:${VENDOR_PORT}/${VENDOR_PATH}:${VERSION}" \
      "localhost:${LOCAL_PORT}/${COMPONENT_PATH}:${VERSION}"
    ;;

  verify)
    require_port_forward
    header "Verify: the image is pullable by digest from the component's own repository"
    digest=$(image_local_reference)
    if [ -z "$digest" ]; then
      echo "no localReference for the image resource -- run transfer and copy first" >&2
      exit 1
    fi
    echo "image localReference:  ${digest}"
    echo "component descriptor:  HTTP $(http_code "$COMPONENT_PATH" "$VERSION")"
    echo "image by that digest:  HTTP $(http_code "$COMPONENT_PATH" "$digest")"
    echo
    echo "A 200 on the second line is the point: the image lives inside the component's"
    echo "repository, so the kubelet can pull it without ever reaching ghcr.io."
    ;;

  deploy)
    require_port_forward
    header "4. Deploy: Context, Target, Installation"
    apply_target
    run "${KUBECTL[@]}" apply -f "$OUT/manifests/context.yaml"

    # A re-run must not adopt the previous Installation's finished job, and the helm
    # release is torn down with it unless we say otherwise.
    if "${KUBECTL[@]}" get installation localblob-demo -n "$NAMESPACE" >/dev/null 2>&1; then
      "${KUBECTL[@]}" annotate installation localblob-demo -n "$NAMESPACE" \
        landscaper.gardener.cloud/delete-without-uninstall=true --overwrite >/dev/null
      run "${KUBECTL[@]}" delete installation localblob-demo -n "$NAMESPACE" \
        --wait=true --timeout=180s || true
    fi
    run "${KUBECTL[@]}" apply -f "$OUT/manifests/installation.yaml"

    settle localblob-demo
    show_status
    ;;

  descriptor)
    require_port_forward
    if [ "$(http_code "$COMPONENT_PATH" "$VERSION")" = "200" ]; then
      header "Component descriptor as delivered -- no registry address anywhere in it"
      run "$OCM" get component-version "${REPO_HOST}//${COMPONENT}:${VERSION}" -o yaml |
        yq '.[0]'
    else
      echo "component not in the in-cluster registry yet; run copy first"
    fi
    ;;

  status)
    show_status
    ;;

  clean)
    header "Clean: k8s resources and generated files; the cluster stays"
    "${KUBECTL[@]}" delete installation localblob-demo -n "$NAMESPACE" \
      --ignore-not-found --timeout=120s || true
    "${KUBECTL[@]}" delete namespace "$NAMESPACE" "$DEPLOY_NAMESPACE" \
      --ignore-not-found --timeout=120s || true
    # Only this experiment's repositories, so a registry shared via KIND_CLUSTER=ls keeps
    # whatever else lives in it.
    for repo in internal vendor; do
      "${KUBECTL[@]}" exec "deploy/${REGISTRY_SVC}" -n "$REGISTRY_NS" -- \
        rm -rf "/var/lib/registry/docker/registry/v2/repositories/${repo}" 2>/dev/null || true
    done
    run rm -rf "$OUT"
    ;;

  ##### the environment #####

  up)
    ensure_ocm
    "$0" cluster
    "$0" images
    "$0" install
    "$0" vendor-registry
    header "Environment ready -- run the experiment with: ./run-experiment.sh all"
    ;;

  cluster)
    if kind get clusters 2>/dev/null | grep -qx "$KIND_CLUSTER"; then
      echo "kind cluster ${KIND_CLUSTER} already exists"
      exit 0
    fi
    header "Create the kind cluster ${KIND_CLUSTER}"
    run kind create cluster --name "$KIND_CLUSTER" --image "$NODE_IMAGE"
    run "${KUBECTL[@]}" wait --for=condition=Ready node --all --timeout=180s
    ;;

  install-ocm)
    bin_dir="$(dirname "$OCM")"
    header "Install the OCM v2 CLI into ${bin_dir}"
    # https://ocm.software/docs/getting-started/ocm-cli-installation/
    # Note install-cli.sh is the v2 installer; install.sh is the legacy v1 one.
    mkdir -p "$bin_dir"
    installer="$OUT/install-cli.sh"
    run curl -sfL https://ocm.software/install-cli.sh -o "$installer"
    run bash "$installer" "$bin_dir"

    header "Verify"
    run "$OCM" version
    # The v1 CLI capitalises the field names ("GitVersion"); v2 prints them lowercase.
    if ! "$OCM" version 2>/dev/null | grep -q '"gitVersion"'; then
      echo "${OCM} is not the v2 CLI" >&2
      exit 1
    fi
    ;;

  images)
    header "Build the workspace binaries and images, load them into kind/${KIND_CLUSTER}"
    arch=$(docker exec "${KIND_CLUSTER}-control-plane" uname -m)
    [ "$arch" = "aarch64" ] && arch="arm64"
    [ "$arch" = "x86_64" ] && arch="amd64"
    echo "node architecture: ${arch}, image tag: ${IMAGE_TAG}"

    for img in $IMAGES; do
      header "Build ${img}"
      ( cd "$REPO_ROOT" && run env CGO_ENABLED=0 GOOS=linux GOARCH="$arch" \
          go build -o "bin/${img}.linux-${arch}" \
          -ldflags "-X github.com/openmcp-project/landscaper/pkg/version.GitVersion=${IMAGE_TAG} \
-X github.com/openmcp-project/landscaper/pkg/version.gitCommit=$(git -C "$REPO_ROOT" rev-parse --verify HEAD)" \
          "cmd/${img}/main.go" )
      ( cd "$REPO_ROOT" && run docker build --platform "linux/${arch}" --target "$img" \
          -t "${img}:${IMAGE_TAG}" -f Dockerfile . )
      run kind load docker-image "${img}:${IMAGE_TAG}" --name "$KIND_CLUSTER"
    done
    ;;

  install|reinstall)
    if [ "$1" = "reinstall" ]; then
      header "Uninstall the existing releases"
      for release in landscaper helm-deployer manifest-deployer; do
        run helm --kube-context "$KUBECTX" -n landscaper uninstall "$release" || true
      done
    fi

    "$0" registry

    header "Install Landscaper ${IMAGE_TAG} from ${REPO_ROOT}/charts"
    cat >"$OUT/values-landscaper.yaml" <<'EOF'
landscaper:
  landscaper:
    registryConfig:
      # the experiment pushes to the in-cluster registry over plain http
      allowPlainHttpRegistries: true
    deployerManagement:
      disable: true
      namespace: landscaper
      agent:
        disable: true
    deployers: []
EOF
    run helm --kube-context "$KUBECTX" upgrade --install landscaper \
      "$REPO_ROOT/charts/landscaper" \
      -n landscaper --create-namespace --take-ownership --force-conflicts \
      -f "$OUT/values-landscaper.yaml" \
      --set landscaper.controller.image.repository=landscaper-controller \
      --set landscaper.controller.image.tag="$IMAGE_TAG" \
      --set landscaper.webhooksServer.image.repository=landscaper-webhooks-server \
      --set landscaper.webhooksServer.image.tag="$IMAGE_TAG"

    header "Install the helm deployer ${IMAGE_TAG}"
    run helm --kube-context "$KUBECTX" upgrade --install helm-deployer \
      "$REPO_ROOT/charts/helm-deployer" \
      -n landscaper --take-ownership --force-conflicts \
      --set deployer.oci.allowPlainHttp=true \
      --set image.repository=helm-deployer-controller \
      --set image.tag="$IMAGE_TAG"

    header "Install the manifest deployer ${IMAGE_TAG}"
    run helm --kube-context "$KUBECTX" upgrade --install manifest-deployer \
      "$REPO_ROOT/charts/manifest-deployer" \
      -n landscaper --take-ownership --force-conflicts \
      --set image.repository=manifest-deployer-controller \
      --set image.tag="$IMAGE_TAG"

    header "Wait for the rollout"
    for deploy in landscaper landscaper-main landscaper-webhooks helm-deployer manifest-deployer; do
      run "${KUBECTL[@]}" -n landscaper rollout status "deployment/${deploy}" --timeout=180s || true
    done
    run "${KUBECTL[@]}" -n landscaper get deployment \
      -o custom-columns=NAME:.metadata.name,IMAGE:.spec.template.spec.containers[0].image
    ;;

  registry)
    header "Install the in-cluster OCI registry"
    # The component is delivered here and the Landscaper resolves it from here, so it needs
    # a registry reachable under a cluster-internal name. It speaks plain http, which is why
    # the chart gets allowPlainHttpRegistries and every reference carries http://. Storage is
    # a PVC on kind's default local-path class so a pod restart keeps the delivered component.
    "${KUBECTL[@]}" get namespace "$REGISTRY_NS" >/dev/null 2>&1 ||
      run "${KUBECTL[@]}" create namespace "$REGISTRY_NS"

    echo "+ kubectl apply -f - (registry deployment, service, pvc)"
    cat <<EOF | "${KUBECTL[@]}" apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ${REGISTRY_SVC}-data
  namespace: ${REGISTRY_NS}
  labels:
    app: ${REGISTRY_SVC}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: ${REGISTRY_SIZE}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${REGISTRY_SVC}
  namespace: ${REGISTRY_NS}
  labels:
    app: ${REGISTRY_SVC}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${REGISTRY_SVC}
  template:
    metadata:
      labels:
        app: ${REGISTRY_SVC}
    spec:
      containers:
        - name: registry
          image: ${REGISTRY_IMAGE}
          ports:
            - containerPort: 5000
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
      volumes:
        - name: registry-data
          persistentVolumeClaim:
            claimName: ${REGISTRY_SVC}-data
---
apiVersion: v1
kind: Service
metadata:
  name: ${REGISTRY_SVC}
  namespace: ${REGISTRY_NS}
  labels:
    app: ${REGISTRY_SVC}
spec:
  selector:
    app: ${REGISTRY_SVC}
  ports:
    - port: 5000
      targetPort: 5000
EOF
    run "${KUBECTL[@]}" -n "$REGISTRY_NS" rollout status "deployment/${REGISTRY_SVC}" --timeout=180s
    "$0" node-config
    ;;

  node-config)
    # Image pulls happen on the node, outside the cluster network, so the service DNS name
    # does not resolve there and the registry speaks plain http. A containerd hosts.toml maps
    # the host to the service ClusterIP and allows http. This experiment needs it: the image
    # is a local blob the kubelet pulls straight from the component's own repository.
    if docker exec "${KIND_CLUSTER}-control-plane" \
         test -f "/etc/containerd/certs.d/${REGISTRY_CLUSTER_HOST}/hosts.toml" 2>/dev/null; then
      echo "containerd already resolves ${REGISTRY_CLUSTER_HOST}"
      exit 0
    fi

    header "Let the node pull from ${REGISTRY_CLUSTER_HOST}"
    cluster_ip=$("${KUBECTL[@]}" -n "$REGISTRY_NS" get "svc/${REGISTRY_SVC}" \
      -o jsonpath='{.spec.clusterIP}')
    echo "+ containerd hosts.toml -> http://${cluster_ip}:5000"
    docker exec "${KIND_CLUSTER}-control-plane" sh -c "
      grep -q 'certs.d' /etc/containerd/config.toml || cat >> /etc/containerd/config.toml <<'TOML'

[plugins.\"io.containerd.cri.v1.images\".registry]
  config_path = \"/etc/containerd/certs.d\"
TOML
      mkdir -p '/etc/containerd/certs.d/${REGISTRY_CLUSTER_HOST}'
      cat > '/etc/containerd/certs.d/${REGISTRY_CLUSTER_HOST}/hosts.toml' <<TOML
server = \"http://${cluster_ip}:5000\"
[host.\"http://${cluster_ip}:5000\"]
  capabilities = [\"pull\", \"resolve\"]
  skip_verify = true
TOML
      systemctl restart containerd"
    run "${KUBECTL[@]}" wait --for=condition=Ready \
      "node/${KIND_CLUSTER}-control-plane" --timeout=180s
    ;;

  vendor-registry)
    if curl -s -o /dev/null -m 1 "http://localhost:${VENDOR_PORT}/v2/"; then
      echo "vendor registry already reachable on localhost:${VENDOR_PORT}"
      exit 0
    fi
    header "Start the vendor registry on localhost:${VENDOR_PORT}"
    # A throwaway container: it only has to hold the component between the transfer and the
    # oras cp, so no volume. Removing it is what "down" does.
    docker rm -f "$VENDOR_CONTAINER" >/dev/null 2>&1 || true
    run docker run -d --name "$VENDOR_CONTAINER" \
      -p "${VENDOR_PORT}:5000" "$REGISTRY_IMAGE"
    for _ in $(seq 1 30); do
      curl -s -o /dev/null -m 1 "http://localhost:${VENDOR_PORT}/v2/" && exit 0
      sleep 1
    done
    echo "vendor registry not reachable on localhost:${VENDOR_PORT}" >&2
    exit 1
    ;;

  down)
    header "Remove the vendor registry"
    run docker rm -f "$VENDOR_CONTAINER" || true

    if kind get clusters 2>/dev/null | grep -qx "$KIND_CLUSTER"; then
      header "Delete the kind cluster ${KIND_CLUSTER}"
      run kind delete cluster --name "$KIND_CLUSTER"
    else
      echo "no kind cluster named ${KIND_CLUSTER}"
    fi
    run rm -rf "$OUT"
    ;;

  all)
    "$0" up
    for step in setup component transfer copy verify deploy; do
      "$0" "$step"
    done
    header "Done -- the pod image above is the component's own repository plus a digest"
    ;;

  *)
    echo "Usage: run-experiment.sh <case>"
    echo ""
    echo "  ./run-experiment.sh all      everything, from nothing"
    echo "  ./run-experiment.sh down     delete the kind cluster and the vendor registry"
    echo ""
    echo "Experiment:  setup, component, transfer, copy, verify, deploy, descriptor,"
    echo "             status, clean"
    echo "Environment: up, cluster, install-ocm, images, install, reinstall, registry,"
    echo "             node-config, vendor-registry, down"
    exit 1
    ;;
esac
