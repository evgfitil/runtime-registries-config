# Runtime Registry Config

This repository contains:

- The application code that adds redirection parameters for container registry requests into the container runtime configuration (specifically for CRI-O).
- A [Helm chart](./chart/runtime-registry-config) for deploying the application within a Kubernetes cluster.

## Operation Principle

The application is deployed in a Kubernetes cluster as a DaemonSet. It requires specific `CAPABILITIES` (CAP_SYS_ADMIN, CAP_SYS_CHROOT) to function properly. It continuously monitors a ConfigMap for changes in the configuration and ensures that the state of the container registry mirrors on the cluster nodes is in accordance with what is declared.

## Getting Started

### Prerequisites

- Kubernetes cluster
- Helm 3.x

### Installation

To install the application, follow these steps:

```bash
helm repo add my-repo http://path-to-your-repo
helm install my-app my-repo/runtime-registry-config
```

## Configuration

Modify the `values.yaml` file in the Helm chart to customize the application behavior. For example, to add registry mirrors, you can specify them as follows:

```yaml
registries:
  - original: "docker.io"
    mirror: "some-private-registry.com"
    insecure: true
```

This configuration sets up a mirror for the Docker Hub (`docker.io`) pointing to a private registry (`some-private-registry.com`) and marks the connection as insecure.

## To-Do

- [ ] Support for containerd CRI
- [ ] Support for image loading under authenticated users
