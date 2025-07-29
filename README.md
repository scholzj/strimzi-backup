# Strimzi Backup

_Note: This is not part of the Strimzi Cloud Native Computing Foundation (CNCF) project!_

Strimzi Backup is a CLI tool for backing up and restoring your [Strimzi-based Apache Kafka cluster](https://strimzi.io).

## How to use Strimzi Backup?

### Installation

You can download one of the release binaries from one of the [GitHub releases](https://github.com/scholzj/strimzi-backup/releases) and use it.
Alternatively, you can also use the provided container image to run it from a Kubernetes Pod or locally as a container.

### Configuration options

Strimzi Backup supports several command line options:

TODO: Fix

| Option             | Description                                                                                                                                                | Default Value |
|--------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| `--help` / `-h`    | Help                                                                                                                                                       |               |
| `--kubeconfig`     | Path to the kubeconfig file to use for Kubernetes API requests. If not specified, `strimzi-shutdown` will try to auto-detect the Kubernetes configuration. |               |
| `--namespace`      | Namespace of the Kafka cluster. If not specified, defaults to the namespace from your Kubernetes configuration.                                            |               |
| `--name`           | Name of the Kafka cluster.                                                                                                                                 |               |
| `--timeout` / `-t` | Timeout for how long to wait for the Proxy Pod to become ready. In milliseconds.                                                                           | `300000`      |

### Backing up your Apache Kafka cluster

TODO: Fix

### Restoring your Apache Kafka cluster

TODO: Fix

### Scheduled backup of your Apache Kafka cluster using Kubernetes `CronJob`

TODO: Fix 

## Frequently Asked Questions

### Does Strimzi Backup support ZooKeeper-based clusters?

No, Strimzi Backup currently supports only KRaft-based Apache Kafka clusters.

### Any plans to support other Strimzi resources?

Currently, the support is planned only for Apache Kafka and Apache Kafka Connect clusters, which consist of multiple custom resources, and (in case of Apache Kafka clusters) use persistent volumes to store data.
The other resources such as Mirror Maker 2 or Bridge are stateless and consist of a single custom resource.
So you can easily back them up with `kubectl get ... -o yaml` and do not need any special tools.
