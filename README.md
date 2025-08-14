# Strimzi Backup

_Note: This is not part of the Strimzi Cloud Native Computing Foundation (CNCF) project!_

Strimzi Backup is a CLI tool for backing up and restoring your [Strimzi-based Apache Kafka cluster](https://strimzi.io).

## How to use Strimzi Backup?

### Installation

You can download one of the release binaries from one of the [GitHub releases](https://github.com/scholzj/strimzi-backup/releases) and use it.
Alternatively, you can also use the provided container image to run it from a Kubernetes Pod or locally as a container.

### Getting help

You can always get help by using the `--help` command 😉.
You can also ask in [discussions](https://github.com/scholzj/strimzi-backup/discussions).

### Backing up your Apache Kafka cluster

You can back up your Kafka cluster using the `strimzi-backup backup kafka` command.
This command will get the Kubernetes resources and store them in a GZIP archive.
It will include:
* The `Kafka` CR
* (Optional) The Secrets with the Cluster and Client Certification Authorities
* All `KafkaNodePool` CRs belonging to this Kafka cluster
* All `KafkaTopic` CRs belonging to this Kafka cluster
* All `KafkaUser` CRs belonging to this Kafka cluster
* (Optional) All Secrets belonging to the Kafka Users with their mTLS or SCRAM-SHA-512 credentials

The backup command uses the following options:

| Option                      | Description                                                                                                                                                                                                                                                                                                                                                                                                   | Default Value |
|-----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| `--kubeconfig`              | Path to the kubeconfig file to use for Kubernetes API requests. If not specified, `strimzi-backup` will try to auto-detect the Kubernetes configuration.                                                                                                                                                                                                                                                      |               |
| `--namespace`               | Namespace of the Kafka cluster to backup. If not specified, `strimzi-backup` will try to auto-detect and use the current namespace from your Kubernetes configuration.                                                                                                                                                                                                                                        |               |
| `--name`                    | Name of the Kafka cluster to backup. (Required)                                                                                                                                                                                                                                                                                                                                                               |               |
| `--filename`                | Name of the file with the backup. If not set, the backup will be _auto-generated_ based on the current time.                                                                                                                                                                                                                                                                                                  |               |
| `--skip-metadata-cleansing` | Skip cleanup of the Kubernetes metadata in the backed up resources. Metadata cleansing removes the fields that are not useful for restoring the cluster such as the generation, timestamps, managed fields, or last applied configurations. Skipping the metadata cleansing will make the resulting backup file larger. But in some cases - for example for auditing purposes - the metadata might be useful. | `false`       |
| `--skip-ca-secrets`         | Skip backup of the Cluster and Client Certification Authority Secrets                                                                                                                                                                                                                                                                                                                                         | `false`       |
| `--skip-user-secrets`       | Skip backup of the Kafka User Secrets                                                                                                                                                                                                                                                                                                                                                                         | `false`       |

Notes:
* The server certificates used by the different nodes are not part of the backup.
  The Strimzi Cluster Operator will just create new ones once the cluster is restored.
  As clients trust the Kafka cluster based on its Cluster CA, restoring the CLuster CA is sufficient to make sure the original trusted certificates work.
* `strimzi-backup` does not include any third party Secrets (such as listener server certificates).
  You are resonsible for backing them up and restoring them yourself.

### Restoring your Apache Kafka cluster

You can restore your Kafka cluster using the `strimzi-backup restore kafka` command.
This command will load the resources from the backup and recreate them.
It will first recreate the Kafka cluster in the paused state, restore its Kafka Cluster ID and only at the end, once all other resources are restored as well, it will unpause it and wait for ti to get ready.
The restore will include all resources fromt he backup file:
* The `Kafka` CR
* The Secrets with the Cluster and Client Certification Authorities
* All `KafkaNodePool` CRs belonging to this Kafka cluster
* All `KafkaTopic` CRs belonging to this Kafka cluster
* All `KafkaUser` CRs belonging to this Kafka cluster
* All Secrets belonging to the Kafka Users with their mTLS or SCRAM-SHA-512 credentials

The restore command uses the following options:

| Option                | Description                                                                                                                                                                                                                                            | Default Value |
|-----------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| `--kubeconfig`        | Path to the kubeconfig file to use for Kubernetes API requests. If not specified, `strimzi-backup` will try to auto-detect the Kubernetes configuration.                                                                                               |               |
| `--namespace`         | Namespace in which the Kafka cluster should be restored. If not specified, `strimzi-backup` will try to auto-detect and use the current namespace from your Kubernetes configuration. This might differ from the original name when the back was done. |               |
| `--name`              | Name of the restored Kafka cluster. This might differ from the original name when the back was done. `strimzi-backup` will rename the cluster accordingly. (Required)                                                                                  |               |
| `--filename`          | Name of the file with the backup which should be restored. (Required)                                                                                                                                                                                  |               |
| `--timeout`           | Timeout for how long to wait for the cluster to restore. In milliseconds.                                                                                                                                                                              | `300000`      |
| `--skip-ca-secrets`   | Skip restoring of the Cluster and Client Certification Authority Secrets                                                                                                                                                                               | `false`       |
| `--skip-user-secrets` | Skip restoring of the Kafka User Secrets                                                                                                                                                                                                               | `false`       |
| `--skip-cluster-id`   | Skip restoring of the Kafka Cluster ID                                                                                                                                                                                                                 | `false`       |

Notes:
* In most cases, Strimzi cannot fully restore the addresses of the external listeners.
  Things such as load balancers will be newly provisioned when the cluster is restored and are likely to differ from the original ones.
* The addresses of the internal listeners will also differ in case you change the namespace or name of the Kafka cluster.
* The restore process expects to do the restoration into a clean environment and will currently fail if any of the resources already exists.
  This might be addressed in the future with the _dry-run_ and _force_ modes (see [#11](https://github.com/scholzj/strimzi-backup/issues/11) for more details).

### Exporting the resources from the backup

You can use the command `strimzi-backup export` command to export the custom resources from the backup archive to separate YAML files.
The export command uses the following options:

| Option               | Description                                                           | Default Value |
|----------------------|-----------------------------------------------------------------------|---------------|
| `--filename`         | Name of the file with the backup which should be exported. (Required) |               |
| `--target-directory` | The directory where the files should be exported. (Required)          |               |

## Future Plans

There are several features I plan to add in the future.
The major ones are:
* Support for Kafka Connect and Connectors
* Support for data backup for Kafka clusters
* Support for backing up into Config Map / Secret to allow running the tool from a CronJob
* Tests 🙄

## Frequently Asked Questions

### Does Strimzi Backup support ZooKeeper-based clusters?

No, Strimzi Backup supports only KRaft-based Apache Kafka clusters.
There are currently no plans to support ZooKeeper-based clusters.

### Any plans to support other Strimzi resources?

Currently, the support is planned only for Apache Kafka and Apache Kafka Connect clusters, which consist of multiple custom resources, and (in case of Apache Kafka clusters) use persistent volumes to store data.
The other resources such as Mirror Maker 2 or Bridge are stateless and consist of a single custom resource.
So you can easily back them up with `kubectl get ... -o yaml` and do not need any special tools.
