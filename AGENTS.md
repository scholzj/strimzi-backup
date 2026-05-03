# AGENTS

## Scope
- This repo is a single Go CLI, not a monorepo. `main.go` only calls `cmd.Execute()`.
- CLI wiring lives in `cmd/`. Real behavior lives in `pkg/backuper`, `pkg/restorer`, `pkg/exporter`, and `pkg/utils`.

## Toolchain And Verification
- Use Go 1.26.x. `go.mod` now requires `go 1.26.0`; CI should install `1.26.x`.
- Fast local smoke check: `go build`.
- CI PR check order is effectively `go build` then `CGO_ENABLED=0 go test -v ./...`.
- There are currently no `*_test.go` files, so `go test ./...` is mostly a compile/regression check.
- There is no repo-local lint, formatter, or typecheck config to run beyond normal Go tooling.

## Runtime Behavior Worth Knowing
- Kubernetes config resolution in `pkg/utils/kubernetes_utils.go` is: `--kubeconfig` -> `KUBECONFIG` -> `~/.kube/config` -> in-cluster config.
- `--namespace` is optional only if the kubeconfig or in-cluster service account provides a namespace; otherwise commands fail.
- Backup output filenames default to `backup-YYYY-MM-DD-HH-MM-SS.gz` and are opened with `os.O_EXCL`, so reusing an existing filename fails instead of overwriting.
- Backup archives are multi-stream gzip files. Each stream uses the gzip header `Name` field (`kafka.yaml`, `kafka-topics.yaml`, etc.) as the logical filename. `pkg/exporter` and restore flows depend on those names.
- Backup metadata cleansing is on by default. `--skip-metadata-cleansing` keeps Kubernetes metadata that restore logic normally strips.

## Command Structure
- `backup kafka` and `restore kafka` handle Strimzi Kafka clusters plus related node pools, topics, users, and optional secrets.
- `backup connect` and `restore connect` handle Kafka Connect clusters plus `KafkaConnector` resources.
- `export` reads a backup archive and writes each gzip stream to a separate YAML file under `--target-directory`.

## Restore-Specific Gotchas
- Kafka restore is intentionally ordered: create Kafka paused, restore dependent resources, restore cluster ID, then unpause and wait for readiness.
- Restore code rewrites namespace and cluster labels to the target `--namespace` / `--name`; do not assume backups are replayed verbatim.
- README and code both indicate restore expects a clean target environment; existing resources cause failures.

## Release / Container Notes
- Release workflow cross-builds binaries named `strimzi-backup-<version>-<os>-<arch>` (plus `.exe` on Windows).
- `Dockerfile` is `FROM scratch` and only `ADD`s a prebuilt binary matching `strimzi-backup-*-${TARGETOS}-${TARGETARCH}`. A plain `docker build` will fail unless that binary already exists in the build context.
