# Plan: On-Demand Backup v2 Implementation

## TL;DR

Implement on-demand (now) backup for v2 API. BackupClass replaces hard-coded DB-type logic (logical/physical for PSMDB, PITR for PG). UI form is universal — BackupClass config schema drives dynamic fields. No dbType checks in UI. Schedules are out of scope (later).

## API Mapping: Old → New

| Action              | Old API                                                                          | New API                                                     |
| ------------------- | -------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Create backup       | POST /namespaces/{ns}/database-cluster-backups                                   | POST /clusters/{cluster}/namespaces/{ns}/backups            |
| Get backup          | GET /namespaces/{ns}/database-cluster-backups/{name}                             | GET /clusters/{cluster}/namespaces/{ns}/backups/{backup}    |
| Delete backup       | DELETE /namespaces/{ns}/database-cluster-backups/{name}?cleanupBackupStorage=... | DELETE /clusters/{cluster}/namespaces/{ns}/backups/{backup} |
| List backups        | GET /namespaces/{ns}/database-clusters/{name}/backups                            | **MOCK** (not in v2 yet)                                    |
| List backup classes | N/A                                                                              | GET /clusters/{cluster}/backup-classes                      |
| Get backup class    | N/A                                                                              | GET /clusters/{cluster}/backup-classes/{backupClass}        |

## Data Model Changes

- `spec.dbClusterName` → `spec.instanceName`
- `spec.backupStorageName` → `spec.destination.backupStorageName` or `spec.destination.s3`
- NEW: `spec.backupClassName` (required) — replaces hard-coded engine logic
- NEW: `spec.config` (optional, validated against BackupClass openAPIV3Schema)
- Status: `created` → `startedAt`, `completed` → `completedAt`
- Old statuses: OK(Succeeded)/Failed/InProgress/Unknown/Deleting
- New statuses: Pending/Running/Succeeded/Failed/Error (no Deleting — track via deletionTimestamp)

## BackupClass Key Facts

- Cluster-scoped CRD, READ-ONLY API (no create/update for users)
- Created by plugin developers / cluster admins as part of provider installation
- `supportedProviders: string[]` — filter by provider when showing in UI
- `config.openAPIV3Schema` — standard JSON Schema for dynamic backup config fields
- `displayName` — human-readable label for UI
- Replaces: PSMDB logical/physical selection, future physical backup support
- If 1 class for provider → auto-select, no field needed
- If multiple → show select with displayName

## DB-Type-Specific Logic Moved to Providers

| Old UI Logic                                | Old Location                                      | v2 Owner                                        |
| ------------------------------------------- | ------------------------------------------------- | ----------------------------------------------- |
| PG: auto-enable PITR on backup create       | on-demand-backup-modal.tsx L73-87                 | ⏳ WAITING: BE answer                           |
| PG: auto-disable PITR on last backup delete | backups-list.tsx L118-135                         | ⏳ WAITING: BE answer                           |
| PG: 3 storage slot limit                    | utils/backups.ts L11-38, backups-list.tsx L96-112 | ⏳ WAITING: BE answer                           |
| PSMDB: logical/physical radio               | on-demand-backup-fields-wrapper.tsx L49           | BackupClass (separate classes or config schema) |
| PG: schedule limit (3 max)                  | backups-list-table-header.tsx                     | ⏳ Out of scope (schedules)                     |

## Implementation Plan

### Phase 1: API Layer (types, functions, hooks)

1. **v2 backup types** — new shared-types file based on Backup schema from crds.gen.types.ts
   - Backup, BackupStatus enum (Pending/Running/Succeeded/Failed/Error)
   - BackupClass types
   - Create payload type
2. **v2 API functions** — api layer
   - createBackup(cluster, namespace, backup)
   - getBackup(cluster, namespace, backupName)
   - deleteBackup(cluster, namespace, backupName)
   - listBackupClasses(cluster)
   - getBackupClass(cluster, backupClassName)
   - listBackups → **MOCK** returning empty list until BE adds endpoint
3. **React Query hooks**
   - useCreateBackup(cluster, namespace)
   - useDeleteBackup(cluster, namespace)
   - useListBackups(cluster, namespace, instanceName) → mock
   - useListBackupClasses(cluster)
4. **RBAC** — check `backups` for create/delete, `backup-classes` for read (separate resources)

### Phase 2: UI — Backups Tab on Instance Details

5. **Add Backups tab** to instance details page (similar to old db-cluster-details/backups)
6. **BackupsList table** with columns:
   - Status (Pending/Running/Succeeded/Failed/Error)
   - Name
   - Backup Class (displayName)
   - Started (startedAt)
   - Completed (completedAt)
   - Actions (delete)
7. **Table header** with "Create backup" dropdown → only "Now" button (schedules later)
8. **Auto-refresh** every 10s (polling, same as v1)

### Phase 3: On-Demand Backup Modal

9. **Universal form** (NO dbType checks):
   - `name` — auto-generated `backup-{shortUID}`, editable
   - `backupClassName` — select from filtered BackupClasses (by supportedProviders matching instance provider)
     - If 1 option → auto-select, possibly hide field
     - If multiple → show select with displayName
   - `destination` — backup storage select (from available storages in namespace)
   - `config` fields — **deferred**: render from BackupClass openAPIV3Schema if present
     - Initial implementation: skip config fields, handle when real BackupClasses exist
     - Future: simple JSON Schema → form renderer (NOT full UI Generator)
10. **Create payload**:
    ```typescript
    {
      metadata: { name: "backup-xxxxx" },
      spec: {
        instanceName: "my-instance",
        backupClassName: "selected-class",
        destination: { backupStorageName: "my-storage" },
        config: { /* from BackupClass schema, if any */ }
      }
    }
    ```

### Phase 4: Delete Backup

11. **Delete confirmation dialog**
12. ⏳ WAITING: cleanupBackupStorage — include cleanup toggle if BE adds parameter

### Phase 5: E2E Tests

13. **PR-level tests** (with mocks):
    - On-demand backup form validation (name, storage, backup class selection)
    - Backup RBAC (backups + backup-classes as separate resources)
14. **Release-level tests** — adapt after full backend is ready

## Pending Questions (sent to BE)

1. ⏳ **cleanupBackupStorage** — will v2 DELETE have this param or auto-cleanup via CleanupJobSpec?
2. ⏳ **PITR auto-enable/disable** — provider controller/webhook responsibility?
3. ⏳ **Storage slot limit** — provider enforcement? DataStoreConstraints? UI warning mechanism?
4. ⏳ **ListBackups endpoint** — timeline for adding to v2 API?

## Files Reference (old → new mapping)

| Purpose       | Old File                          | New File (to create)                  |
| ------------- | --------------------------------- | ------------------------------------- |
| Types         | shared-types/backups.types.ts     | TBD (v2 types)                        |
| API functions | api/backups.ts                    | TBD (v2 API)                          |
| Hooks         | hooks/api/backups/useBackups.ts   | TBD (v2 hooks)                        |
| Backup tab    | pages/db-cluster-details/backups/ | pages/instance-details/backups/ (TBD) |
| Modal         | .../on-demand-backup-modal/       | TBD (universal modal)                 |
| Table         | .../backups-list/                 | TBD                                   |
| Constants     | .../backups-list.constants.ts     | TBD (new status mapping)              |

## E2E Tests to Restore/Adapt

| Test                      | File                                               | Priority            |
| ------------------------- | -------------------------------------------------- | ------------------- |
| On-demand form validation | .e2e/pr/db-cluster-details/on-demand-backup.e2e.ts | P1                  |
| Backup RBAC               | .e2e/pr/rbac/backups.e2e.ts                        | P1                  |
| Demand backup full cycle  | .e2e/release/demand-backup.e2e.ts                  | P2 (after BE ready) |
| RBAC demand backup        | .e2e/release/rbac/demand-backup.e2e.ts             | P2                  |
| Backup storage settings   | .e2e/pr/settings/backup-storage.e2e.ts             | P3 (unchanged)      |
| PG backup API test        | api-tests/tests/pg/backup.spec.ts                  | P2 (rewrite for v2) |
