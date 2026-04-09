apiVersion: landscaper.gardener.cloud/v1alpha1
kind: Installation
metadata:
  name: job-example
  namespace: ${namespace}
  annotations:
    landscaper.gardener.cloud/operation: reconcile

spec:

  imports:
    targets:
      - name: cluster        # name of an import parameter of the blueprint
        target: my-cluster   # name of the Target custom resource containing the kubeconfig of the target cluster
    data:
      - name: first                # name of an import parameter of the blueprint
        dataRef: my-first-import   # name of a DataObject containing the parameter value
      - name: second               # name of an import parameter of the blueprint
        dataRef: my-second-import  # name of a DataObject containing the parameter value

  exports:
    data:
      - name: result
        dataRef: my-export

  blueprint:
    inline:
      filesystem:
        blueprint.yaml: |
          apiVersion: landscaper.gardener.cloud/v1alpha1
          kind: Blueprint
          jsonSchema: "https://json-schema.org/draft/2019-09/schema"

          imports:
            - name: cluster
              type: target
              targetType: landscaper.gardener.cloud/kubernetes-cluster
            - name: first
              type: data
              schema:
                type: string
            - name: second
              type: data
              schema:
                type: string

          exports:
            - name: result
              type: data
              schema:
                type: string
          
          deployExecutions:
            - name: default
              type: GoTemplate
              template: |
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
                        - policy: manage
                          manifest:
                            apiVersion: v1
                            kind: ServiceAccount
                            metadata:
                              name: my-job-sa
                              namespace: example
                        - policy: manage
                          manifest:
                            apiVersion: rbac.authorization.k8s.io/v1
                            kind: Role
                            metadata:
                              name: my-job-role
                              namespace: example
                            rules:
                              - apiGroups:
                                  - ""
                                resources:
                                  - configmaps
                                verbs:
                                  - get
                                  - create
                                  - update
                                  - patch
                        - policy: manage
                          manifest:
                            apiVersion: rbac.authorization.k8s.io/v1
                            kind: RoleBinding
                            metadata:
                              name: my-job-rolebinding
                              namespace: example
                            subjects:
                              - kind: ServiceAccount
                                name: my-job-sa
                                namespace: example
                            roleRef:
                              kind: Role
                              name: my-job-role
                              apiGroup: rbac.authorization.k8s.io

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
                      readinessChecks:
                        disableDefault: true
                        custom:
                          - name: JobCompleted
                            resourceSelector:
                              - apiVersion: batch/v1
                                kind: Job
                                name: my-job
                                namespace: example
                            requirements:
                              - jsonPath: .status.succeeded
                                operator: ==
                                values:
                                  - value: 1
                      exports:
                        exports:
                          - key: result
                            fromResource:
                              apiVersion: v1
                              kind: ConfigMap
                              name: my-exports
                              namespace: example
                            jsonPath: .data.result

          exportExecutions:
            - name: my-export-execution
              type: GoTemplate
              template: |
                exports:
                  result: {{ index .deployitems "job-deploy-item" "result" }}
