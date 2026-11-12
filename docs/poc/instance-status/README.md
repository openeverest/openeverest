# Instance status proof of concept: screenshots

These images are evidence for the pull request that proposes Instance status as
a coarse phase plus conditions. They exist so the pull request description can
show them. **Drop this directory before merging.**

Every image except `fixtures-list-dark.png` was captured from the UI of a k3d
cluster running this branch with the Percona Server for MongoDB provider.
`fixtures-list-dark.png` is rendered from the shared display fixtures
(`pkg/instancestatus/testdata/display.json`), not from a cluster.

| File | What it shows |
|---|---|
| `list-all-states.png` | Databases list with every tone at once |
| `list-restoring.png` | The same list while one Instance is seeded from a backup |
| `details-ready-expanded.png` | A ready Instance with the checks and raw conditions expanded |
| `details-creating.png` | First creation |
| `details-upgrading.png` | Rolling version upgrade that keeps the database up |
| `details-seeding-from-backup.png` | New Instance loading data from a backup |
| `details-restoring-in-place.png` | In-place restore of a running Instance |
| `details-crash-loop-early.png` | Crash loop before the problem deadline |
| `details-crash-loop-stalled.png` | The same crash loop after the deadline |
| `details-invalid-configuration.png` | Spec rejected by the provider |
| `details-secret-not-found.png` | Referenced Secret missing |
| `details-backup-storage-not-found.png` | Referenced BackupStorage missing |
| `details-unschedulable.png` | Pods that cannot be scheduled |
| `details-paused-outside.png` | Database paused on the operator's object, outside OpenEverest |
| `details-not-picked-up.png` | No provider reconciles the Instance |
| `details-deleting.png` | Deletion in progress |
| `fixtures-list-dark.png` | Dark theme, with states not reproduced on the cluster |
