---
title: Deploying a Job with the Manifest Deployer
sidebar_position: 5
---

# Deploying a Job with the Manifest Deployer

This example demonstrates how to use the Manifest Deployer to deploy a Kubernetes Job. 

The Installation in this example has import parameter, which are passed to the Job in a mounted Secret. 
The Job writes a ConfigMap, whose data are then exported by the Installation.

For prerequisites, see [here](../../README.md).

## Overview

The Installation deploys the following resources to the target cluster via two DeployItems of type
`landscaper.gardener.cloud/kubernetes-manifest`:

1. **Secret** (`my-imports`) — holds two key-value pairs, namely the import parameters of the Installation.
2. **ServiceAccount** (`my-job-sa`) — the identity under which the Job's pod runs.
3. **Role** (`my-job-role`) — grants permission to create and update ConfigMaps in the `example` namespace.
4. **RoleBinding** (`my-job-rolebinding`) — binds the Role to the ServiceAccount.
5. **Job** (`my-job`) — mounts the Secret as a volume at `/etc/secret`, runs a `bitnami/kubectl` container, and creates a ConfigMap from the values of the imported Secret.

The relevant section of the blueprint looks like this:

```yaml
 deployItems:
    - name: preparation-deploy-item
      type: landscaper.gardener.cloud/kubernetes-manifest
      target:
         import: cluster
      config:
         apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
         kind: ProviderConfiguration
         updateStrategy: update
         manifests:
            - policy: manage
              manifest:
                 apiVersion: v1
                 kind: Secret
                 metadata:
                    name: my-imports
                    namespace: example
                 stringData:
                    first: {{ .imports.first }}
                    second: {{ .imports.second }}

    - name: job-deploy-item
      type: landscaper.gardener.cloud/kubernetes-manifest
      dependsOn:
         - preparation-deploy-item
      target:
         import: cluster
      config:
         apiVersion: manifest.deployer.landscaper.gardener.cloud/v1alpha2
         kind: ProviderConfiguration
         updateStrategy: update
         manifests:
            - policy: manage
              manifest:
                 apiVersion: batch/v1
                 kind: Job
                 metadata:
                    name: my-job
                    namespace: example
                 spec:
                    ttlSecondsAfterFinished: 30
                    template:
                       spec:
                          restartPolicy: Never
                          serviceAccountName: my-job-sa
                          containers:
                             - name: kubectl-container
                               image: bitnami/kubectl:latest
                               command:
                                  - sh
                                  - -c
                                  - >-
                                     kubectl create configmap my-exports --namespace=example
                                     --from-literal=result="$(cat /etc/secret/first) $(cat /etc/secret/second)"
                                     --dry-run=client
                                     -o yaml
                                     | kubectl apply -f -
                               volumeMounts:
                                  - name: secret-volume
                                    mountPath: /etc/secret
                                    readOnly: true
                          volumes:
                             - name: secret-volume
                               secret:
                                  secretName: my-imports
```


## Procedure

1. In the [settings](commands/settings) file, set the variables `RESOURCE_CLUSTER_KUBECONFIG_PATH`
   and `TARGET_CLUSTER_KUBECONFIG_PATH`.

2. On the Landscaper resource cluster, create a namespace `cu-example`.

3. On the target cluster, create a namespace `example`.

4. Run the script [commands/deploy-k8s-resources.sh](commands/deploy-k8s-resources.sh).
   It templates the Target, two DataObjects and the Installation, and applies both to the resource cluster.

5. Wait until the Installation is in phase `Succeeded`.

6. On the resource cluster, verify that the Installation succeeds. Check that a DataObject with the export parameter was created. The result value should be `hello world`, i.e. the import values concatenated, separated by a space.

   ```bash
   kubectl get installation job-example -n example
   kubectl get dataobject my-export -n example -o yaml
   ```

## Cleanup

Remove the Installation with the [delete-installation script](commands/delete-installation.sh).
The Manifest Deployer will delete all managed resources (Secret, ServiceAccount, Role, RoleBinding, Job)
from the target cluster.

Once the Installation is gone, delete the Target with the
[delete-other-k8s-resources script](commands/delete-other-k8s-resources.sh).