# kubectl-parallel-drain

`kubectl-parallel-drain` is a kubectl plugin designed to cordon and drain multiple Kubernetes nodes in parallel. It is particularly useful for cluster upgrades and maintenance tasks where you need to evacuate workloads from a fleet of nodes efficiently.

## Why use this tool?

*   **Parallel Execution**: Unlike standard `kubectl drain` which processes one node at a time, this tool facilitates draining multiple nodes concurrently, significantly checking upgrade times for large clusters.
*   **Continuous Reconciliation**: The tool runs in a continuous loop. It monitors the cluster for nodes matching the specified criteria. If a node is missed, fails to drain, or is accidentally uncordoned, the tool will automatically attempt to cordon and drain it again.
*   **Version Targeting**: Specifically targets nodes running an "old" Kubernetes version, making it safer to use during rolling upgrades.

## Installation

### Prerequisites
*   Go 1.25+ installed

### Steps

1.  Clone the repository:
    ```bash
    git clone https://github.com/oussamarouabah/parallel-drain.git
    cd parallel-drain
    ```

2.  Build the binary:
    ```bash
    go build -o kubectl-parallel_drain main.go
    ```

3.  Install the plugin (move to your PATH):
    ```bash
    sudo mv kubectl-parallel_drain /usr/local/bin/
    ```

4.  Verify installation:
    ```bash
    kubectl plugin list
    # You should see "kubectl-parallel_drain" in the output
    ```

## Usage

Use the command via `kubectl`:

```bash
kubectl parallel-drain --help
```

### Examples

**Basic Usage:**
Drain all nodes running version `v1.32.0`:
```bash
kubectl parallel-drain --old-k8s-version v1.32.0
```

**High Concurrency:**
Drain 5 nodes in parallel:
```bash
kubectl parallel-drain --old-k8s-version v1.32.0 --concurrency 5
```

**Target Specific Nodes:**
Drain nodes in a specific zone running the old version:
```bash
kubectl parallel_drain --old-k8s-version v1.32.0 --selector "topology.kubernetes.io/zone=us-west-2a"
```

**Custom Check Interval:**
Check for cordoned/undrained nodes every 30 seconds:
```bash
kubectl parallel_drain --old-k8s-version v1.32.0 --interval 30s
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--old-k8s-version` | **(Required)** The Kubernetes version to match and drain (e.g., `v1.25.0`). | |
| `--concurrency`, `-c` | Number of nodes to drain in parallel. | `2` |
| `--selector`, `-l` | Label selector to filter nodes (standard kubectl selector). | `""` |
| `--interval` | Duration to wait between checks/reconciliation loops. | `10s` |
