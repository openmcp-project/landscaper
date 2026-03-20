---
title: Prerequisites
sidebar_position: 1
---

# Guided Tour

## Prerequisites and Basic Definitions

- For all examples, you need a [running Landscaper instance](../installation/install-landscaper-controller.md).

- A convenient tool we will often use in the following examples is the [Landscaper
  CLI](https://github.com/openmcp-project/landscapercli). 

- For building reusable components you require the [OCM CLI](https://ocm.software/docs/getting-started/prerequisites).

- During the following exercises, you might need to change files, provided with the examples. For this, you should
  simply clone [this repository](https://github.com/gardener/landscaper) and do the required changes on your local files. You could also fork the repo and work on your fork.

- In all examples, 3 Kubernetes clusters are involved:

  - the **Landscaper Host Cluster**, on which the Landscaper runs
  - the **target cluster**, on which the deployments will be done
  - the **Landscaper Resource Cluster**, on which the various custom resources are stored. These custom resources are
    watched by the Landscaper, and define which deployments should happen on which target cluster.

  It is possible that some or all of these clusters coincide, e.g. in the most simplistic approach, you have only one
  cluster. Such a "one-cluster-setup" is the easiest way to start working with the Landscaper.

## How to follow the Tour

The Guided Tour consists of several chapters, some of which build on each other. In order to get the most out of it, 
you should be following the Guided Tour in this sequence. If you have problems with the presentation of the Guided Tour,
jump to the [original github repository](https://github.com/gardener/landscaper/tree/master/docs/guided-tour).

### A Hello World Example

[1. Hello World Example](./hello-world)

### Basics

[2. Upgrading the Hello World Example](./basics/upgrade)

[3. Manifest Deployer Example](./basics/manifest-deployer)

[4. Multiple Deployments in One Installation](./basics/multiple-deployitems)

### Recovering from Errors

[5. Handling an Immediate Error](./error-handling/immediate-error)

[6. Handling a Timeout Error](./error-handling/timeout-error)

[7. Handling a Delete Error](./error-handling/delete-error)

You can find a list of error messages and corresponding solutions in
[Common Issues](../troubleshooting/common-issues.md).

### Blueprints and Components

[8. Blueprint and Helm Chart Resources in a Component Version](./components/helm-chart)

### Imports and Exports

[9. Import Parameters](./import-export/import-parameters)

[10. Import Data Mappings](./import-export/import-data-mappings)

[11. Export Parameters](./import-export/export-parameters)

[12. Deploying a Job with the Manifest Deployer](./import-export/job)

### Subinstallations

[13. Data flow between Subinstallations](./subinstallations/export-import)

[14. Reuse of Components](./subinstallations/component-references)

[15. Subinstallation Templates](./subinstallations/subinst-templates)

### Templating

[16. Templating: Accessing Component Descriptors ](./templating/components)

### Target Maps

[17. Target Maps: Multiple Deploy Items](./target-maps/01-multiple-deploy-items)

[18. Target Maps: Target Map References](./target-maps/02-targetmap-ref)

[19. Target Maps: Multiple Subinstallations](./target-maps/03-multiple-subinst)

[20. Target Maps: Target Map on Subinstallation Level](./target-maps/04-forward-map)

[21. Target Maps: Other Target Map Examples](./target-maps/05-other-examples)

### Kustomize

[22. Introduction to Kustomize Examples](./kustomize/01-kustomize-introduction)

[23. Deploying the PodInfo Application with Landscaper and Flux](./kustomize/02-podinfo)

[24. Deploying the PodInfo Application with Landscaper and Flux - controlled by Landscaper](./kustomize/03-podinfo-landscaper-controlled)

[25. Dataflow Example](./kustomize/04-dataflow)

### Optimization

[26. Optimization Hints ](../usage/Optimization.md)
