# kubectl-tree

A `kubectl` plugin that displays the resource tree (ownership hierarchy) for Kubernetes objects.

It helps DevOps engineers and SREs visualize the relationships between Deployments, ReplicaSets, and Pods by parsing `ownerReferences` directly from the Kubernetes API.

##  Features

- Visualizes ownership hierarchy (Deployment → ReplicaSet → Pod)
- Fetches real-time status (Ready replicas, Pod phase)
- Works with any namespace (`-n` / `--namespace`)
- Fast and lightweight (single Go binary, no external dependencies)
- Native `kubectl` plugin experience

## Installation

### Option1: Via Krew (Recommended)
`kubectl krew install owntree`

### Option 2: Build from source
# Clone the repository
`git clone https://github.com/nmarakushev/kubectl-tree.git`
cd kubectl-tree

# Build
`make build`

# Build multiplatform
`make build-all`

# Install to /usr/local/bin
`make install`

# Verify installation
`kubectl plugin list`

# Run all tests
`make test`

# Run tests with coverage
`go test -v -cover ./...`