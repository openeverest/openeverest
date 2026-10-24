# Core e2e test provider

This provider uses the runtime from the same OpenEverest commit as the core. It
creates one `pause` pod per Instance and reports Ready when that pod is ready.
Backups write a small marker to the configured S3 bucket. Restores read the
marker. Deleting a Backup removes or retains its object according to the
Backup's deletion policy. Point-in-time restore selects the newest marker at
or before the requested time; it does not emulate a database log stream.

Set `test.openeverest.io/control: fail` or `wait` on an Instance, Backup, or
Restore to exercise predictable failure or waiting states.

The release workflow builds the provider image from the checked-out commit,
loads it into kind, applies `manifests/provider.yaml`, and runs `smoke.sh` after
SeaweedFS is ready. The smoke uses `kubectl` and the AWS CLI on the runner. It
checks Instance readiness, an explicit failure, S3 backup and restore, seeding
a second Instance, Delete/Retain policies, and cascade deletion. The release Playwright project
keeps one PXC PITR golden for real database behavior.
