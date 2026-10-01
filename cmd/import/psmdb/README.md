# PSMDB backup-import discovery job

Self-contained discovery image for Percona Backup for MongoDB (PBM) backups. I

## Setup (once)

Build the image, side-load it into your cluster (no registry push), and install
the import-only `BackupClass` that points at it. Run from the openeverest module
root:

```sh
make -f cmd/import/psmdb/Makefile docker-build
k3d image import openeverest-import-psmdb:dev -c everest-dev
kubectl apply -f cmd/import/psmdb/backupclass.yaml
```

## Usage

Create a `BackupImport` CR referencing the import `BackupClass` and the
`BackupStorage` to discover. The controller spawns the discovery Job; you do
nothing else.

```sh
kubectl apply -f backupimport.yaml
```

Watch progress and the backups it produced:

```sh
kubectl -n everest get backupimport import-from-s3 -w
kubectl -n everest get backups -l backup.openeverest.io/backup-import=import-from-s3
```
