# V2 Backup / Restore / PITR — Architecture

## Table of Contents

- [Goals](#goals)
- [Non-goals](#non-goals)
- [Architecture Decisions](#architecture-decisions)
  - [One BackupClass per provider](#one-backupclass-per-provider)
  - [Three separate config schemas](#three-separate-config-schemas)
  - [Hybrid rendering: static React + dynamic UIGenerator](#hybrid-rendering-static-react--dynamic-uigenerator)
  - [PITR is a per-storage property](#pitr-is-a-per-storage-property)
- [CRD Data Model](#crd-data-model)
  - [BackupClass](#backupclass)
  - [Instance.spec.backup](#instancespecbackup)
  - [Backup CR](#backup-cr)
  - [Restore CR](#restore-cr)
  - [BackupStorage CR](#backupstorage-cr)
- [Backend Changes](#backend-changes)
  - [ProviderManagedSpec extension](#providermanagedspec-extension)
  - [CRD / client regeneration](#crd--client-regeneration)
  - [Provider: BackupClass population](#provider-backupclass-population)
  - [Provider: config consumption](#provider-config-consumption)
- [UI Component Architecture](#ui-component-architecture)
  - [Component tree](#component-tree)
  - [Static vs dynamic fields](#static-vs-dynamic-fields)
  - [UIGenerator integration pattern](#uigenerator-integration-pattern)
- [User Flows](#user-flows)
  - [Flow 1: Create DB with scheduled backups (wizard)](#flow-1-create-db-with-scheduled-backups-wizard)
  - [Flow 2: Create on-demand backup](#flow-2-create-on-demand-backup)
  - [Flow 3: Create schedule — no storages configured yet](#flow-3-create-schedule--no-storages-configured-yet)
  - [Flow 4: Create schedule — storage already exists](#flow-4-create-schedule--storage-already-exists)
  - [Flow 5: Restore from backup](#flow-5-restore-from-backup)
- [Mock Data](#mock-data)
  - [Mock BackupClass with uiSchema](#mock-backupclass-with-uischema)
  - [Mock hook strategy](#mock-hook-strategy)
- [Implementation Phases](#implementation-phases)
  - [Phase 0: Backend — CRD extension + provider support](#phase-0-backend--crd-extension--provider-support)
  - [Phase 1: Foundation — mocks + UISchema support](#phase-1-foundation--mocks--uischema-support)
  - [Phase 2: On-demand backup modal enhancement](#phase-2-on-demand-backup-modal-enhancement)
  - [Phase 3: Storage management](#phase-3-storage-management)
  - [Phase 4: Scheduled backups](#phase-4-scheduled-backups)
  - [Phase 5: Restore](#phase-5-restore)
  - [Phase 6: Instance creation wizard — backups step](#phase-6-instance-creation-wizard--backups-step)
  - [Phase 7: Cleanup](#phase-7-cleanup)
- [File Inventory](#file-inventory)
- [Verification Criteria](#verification-criteria)

---

## Goals

- Dynamic, provider-agnostic backup/restore UI driven by BackupClass schemas
- Shared presentational components (StorageCard, ScheduleFormFields, PITRConfigFields) between wizard and Backups tab; orchestration layer is separate (form-state vs API)
- Graceful degradation when BackupClass has no `uiSchema` (shows only static fields)
- Two-level storage model (NS-level BackupStorage + instance-level binding) is **hidden from the user** — UI presents a single "pick a storage" flow
- Migration of BackupStorage hooks from v1 to v2 API
- BE: extend `ProviderManagedSpec` with `uiSchema` + `storageLimits`
- BE: providers populate BackupClass with config schemas and ui-generator sections
- BE: providers consume `Backup.spec.config`, `Restore.spec.config`, `pitr.config`
- UIGenerator does not require adaptation — it already renders only fields without section chrome

## Non-goals

- Job-mode BackupClass support (only ProviderManaged mode)
- Multi-cluster backup orchestration
- Backup policy / retention policy management (beyond per-schedule retention copies)
- RBAC implementation (re-enabled in cleanup phase)

---

## Architecture Decisions

### One BackupClass per provider

One BackupClass per provider with `backupType` as a field inside `Backup.spec.config` (not separate classes like `psmdb-logical`, `psmdb-physical`).

`Instance.spec.backup.classRef` is a single reference — one class per instance. Backup type is a per-backup decision, not per-instance.

### Three separate config schemas

| Schema         | Stored in                                      | Validated against                                                        | Purpose                                     |
| -------------- | ---------------------------------------------- | ------------------------------------------------------------------------ | ------------------------------------------- |
| Backup config  | `Backup.spec.config`                           | `BackupClass.spec.config.openAPIV3Schema`                                | Per-backup: type, provider-specific options |
| Restore config | `Restore.spec.config`                          | `BackupClass.spec.restoreConfig.openAPIV3Schema`                         | Per-restore: future selective restore       |
| PITR config    | `Instance.spec.backup.storages[N].pitr.config` | `BackupClass.spec.providerManaged.pitrConfig.openAPIV3Schema` (proposed) | Continuous PITR tuning (provider-specific)  |

No shared base needed:

- `backupType` (logical/physical) is **only** in backup config — restore type is auto-derived from source backup
- `compressionType`/`compressionLevel` appear in both backup and PITR but for different purposes (backup data vs oplog chunks)
- PITR target (date/latest) is already first-class in `Restore.spec.dataSource.pitr`
- Each schema has a different lifecycle

### Hybrid rendering: static React + dynamic UIGenerator

```
Static React (hardcoded)              Dynamic (UIGenerator from BackupClass)
───────────────────────────           ──────────────────────────────────────
• PITR toggle (SwitchInput)           • PITR config fields — section="pitr"
• Storage select                      • Backup config fields — section="backup"
• Schedule cron picker                • Restore config fields — section="restore"
• Backup name, restore source
• Modal open/close, date picker
```

Backup/restore forms and modals are universal regardless of provider — the only variable part is provider-specific config fields. On the current iteration we hardcode the static layout and embed UIGenerator for dynamic fields.

**Future UIGenerator extensions** (out of scope for this iteration, but planned):

- Conditional rendering (show/hide based on field value — CEL condition rendering TODO in code)
- New preset uiTypes: `datePicker`, `cronPicker`
- Custom component / modal trigger support (for fieldArray-like patterns)
- Plugin system for provider-defined custom widgets

**Pattern:** Static React modal embeds `<UIGenerator sectionKey="..." />` for provider-specific sub-form. UIGenerator renders only fields (no section title/description chrome) — no adaptation needed.

### PITR is a per-storage property

V1: instance-level toggle. V2: **per-storage** (`Instance.spec.backup.storages[N].pitr`).

- PSMDB: max 1 PITR-enabled storage (enforced via `storageLimits.maxWithPITR`)
- PostgreSQL: may support PITR on all storages
- PITR toggle lives in `<PITRConfigModal />` — opened from the storage card's [Configure PITR] button
- The toggle is disabled when `maxWithPITR` limit is reached for this instance
- When enabled, UIGenerator renders `pitr` section below for provider-specific config

> **Two levels of storage:**
>
> - **BackupStorage CR** (namespace-level): S3 credentials, bucket, endpoint. No PITR config here.
> - **Instance.spec.backup.storages[]** (instance-level): registration of a BackupStorage on a specific instance + PITR toggle + schedules. PITR toggle belongs here.

---

## CRD Data Model

### BackupClass

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupClass
metadata:
  name: psmdb-managed
spec:
  displayName: "Percona Server for MongoDB Backups"
  description: "Physical and logical backups via PBM"
  executionMode: ProviderManaged
  supportedProviders:
    - percona-server-mongodb

  providerManaged:
    supportsPITR: true
    storageLimits:
      max: 3 # max storages per instance
      maxWithPITR: 1 # max PITR-enabled storages
      maxSchedulesPerStorage: 10
    uiSchema: # ui-generator DSL, RawExtension
      backup: # → On-demand backup modal
        label: "Backup Configuration"
        componentsOrder: ["type", "compressionType", "compressionLevel"]
        components:
          type:
            uiType: select
            label: "Backup Type"
            path: type
            required: true
            fieldParams:
              options:
                - { label: "Logical", value: "logical" }
                - { label: "Physical", value: "physical" }
              defaultValue: "logical"
          compressionType:
            uiType: select
            label: "Compression"
            path: compressionType
            fieldParams:
              options:
                - { label: "None", value: "none" }
                - { label: "Gzip", value: "gzip" }
                - { label: "Snappy", value: "snappy" }
                - { label: "LZ4", value: "lz4" }
                - { label: "Zstandard", value: "zstd" }
              defaultValue: "snappy"
          compressionLevel:
            uiType: number
            label: "Compression Level"
            path: compressionLevel
            fieldParams: { defaultValue: 6 }
            validation: { minimum: 0, maximum: 22 }
      pitr: # → Instance storage PITR config sub-form
        label: "PITR Configuration"
        componentsOrder: ["oplogSpanMin", "compressionType"]
        components:
          oplogSpanMin:
            uiType: number
            label: "Oplog Span (minutes)"
            path: oplogSpanMin
            tooltip: "Interval between oplog chunk boundaries"
            fieldParams: { defaultValue: 10 }
            validation: { minimum: 1 }
          compressionType:
            uiType: select
            label: "Oplog Compression"
            path: compressionType
            fieldParams:
              options:
                - { label: "None", value: "none" }
                - { label: "Snappy", value: "snappy" }
                - { label: "Zstandard", value: "zstd" }
              defaultValue: "snappy"
      restore: # → Restore modal dynamic section
        label: "Restore Configuration"
        components: {} # empty for PSMDB initially
```

### Instance.spec.backup

```yaml
spec:
  backup:
    enabled: true
    classRef:
      name: psmdb-managed
    storages:
      - name: s3-main
        storageRef:
          name: my-s3-storage
        main: true
        pitr:
          enabled: true
          config:
            oplogSpanMin: 10
            compressionType: snappy
        schedules:
          - name: daily-full
            cron: "0 2 * * *"
            retentionCopies: 7
            enabled: true
      - name: s3-archive
        storageRef:
          name: my-archive-storage
        main: false
        pitr:
          enabled: false
        schedules:
          - name: monthly-archive
            cron: "0 4 1 * *"
            retentionCopies: 12
            enabled: true
```

### Backup CR

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: Backup
metadata:
  name: my-db-backup-2026-05-11
spec:
  instanceName: my-db
  backupClassName: psmdb-managed
  storageName: s3-main
  config: # from UIGenerator backup section
    type: logical
    compressionType: snappy
    compressionLevel: 6
status:
  state: Succeeded
  size: "2.1Gi"
```

### Restore CR

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: Restore
metadata:
  name: restore-from-backup-001
spec:
  instanceName: my-db
  dataSource:
    backupName: my-db-backup-2026-05-11
    pitr: # optional, only if PITR restore
      type: date
      date: "2026-05-11T01:30:00Z"
  config: {} # from UIGenerator restore section
status:
  state: Succeeded
```

### BackupStorage CR

```yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupStorage
metadata:
  name: my-s3-storage
  namespace: my-ns
spec:
  type: s3
  s3:
    bucket: my-backup-bucket
    region: us-east-1
    endpointURL: https://s3.amazonaws.com
    credentialsSecretName: s3-creds-secret
    verifyTLS: true
    forcePathStyle: false
```

---

## Backend Changes

### ProviderManagedSpec extension

Current state in `api/backup/v1alpha1/backupclass_types.go`:

```go
type ProviderManagedSpec struct {
    SupportsPITR bool `json:"supportsPITR,omitempty"`
}
```

Required change:

```go
type ProviderManagedSpec struct {
    SupportsPITR  bool                  `json:"supportsPITR,omitempty"`
    StorageLimits *StorageLimitsSpec    `json:"storageLimits,omitempty"`
    UISchema      *runtime.RawExtension `json:"uiSchema,omitempty"`
    // PITRConfig schema for Instance.spec.backup.storages[N].pitr.config validation
    PITRConfig    BackupClassConfig     `json:"pitrConfig,omitempty"`
}

type StorageLimitsSpec struct {
    // Max is the maximum number of storages per instance.
    Max *int32 `json:"max,omitempty"`
    // MaxWithPITR is the maximum number of PITR-enabled storages.
    MaxWithPITR *int32 `json:"maxWithPITR,omitempty"`
    // MaxSchedulesPerStorage is the maximum number of schedules per storage.
    MaxSchedulesPerStorage *int32 `json:"maxSchedulesPerStorage,omitempty"`
}
```

### CRD / client regeneration

After modifying `ProviderManagedSpec`:

1. `make generate` — regenerates CRD OpenAPI in `api/openapi/crds.gen.yaml`
2. `make generate-client` — regenerates `client/everest-client.gen.go`
3. No server handler changes — handlers pass through K8s objects, new fields flow automatically
4. No new HTTP endpoints — BackupClass is already read-only (`GET /clusters/{cluster}/backup-classes`)

### Provider: BackupClass population

Each provider must create BackupClass CR(s) at install time or via Helm chart. Example for PSMDB provider:

```yaml
# charts/provider-percona-server-mongodb/templates/backup-class.yaml
apiVersion: backup.openeverest.io/v1alpha1
kind: BackupClass
metadata:
  name: psmdb-managed
spec:
  executionMode: ProviderManaged
  supportedProviders:
    - percona-server-mongodb
  providerManaged:
    supportsPITR: true
    storageLimits:
      max: 3
      maxWithPITR: 1
      maxSchedulesPerStorage: 10
    uiSchema:
      backup:
        label: "Backup Configuration"
        componentsOrder: ["type", "compressionType", "compressionLevel"]
        components:
          type:
            uiType: select
            label: "Backup Type"
            path: type
            required: true
            fieldParams:
              options:
                - { label: "Logical", value: "logical" }
                - { label: "Physical", value: "physical" }
              defaultValue: "logical"
          compressionType:
            uiType: select
            label: "Compression"
            path: compressionType
            fieldParams:
              options:
                - { label: "None", value: "none" }
                - { label: "Gzip", value: "gzip" }
                - { label: "Snappy", value: "snappy" }
                - { label: "LZ4", value: "lz4" }
                - { label: "Zstandard", value: "zstd" }
              defaultValue: "snappy"
          compressionLevel:
            uiType: number
            label: "Compression Level"
            path: compressionLevel
            fieldParams: { defaultValue: 6 }
            validation: { minimum: 0, maximum: 22 }
      pitr:
        label: "PITR Configuration"
        componentsOrder: ["oplogSpanMin", "compressionType"]
        components:
          oplogSpanMin:
            uiType: number
            label: "Oplog Span (minutes)"
            path: oplogSpanMin
            tooltip: "Interval between oplog chunk boundaries"
            fieldParams: { defaultValue: 10 }
            validation: { minimum: 1 }
          compressionType:
            uiType: select
            label: "Oplog Compression"
            path: compressionType
            fieldParams:
              options:
                - { label: "None", value: "none" }
                - { label: "Snappy", value: "snappy" }
                - { label: "Zstandard", value: "zstd" }
              defaultValue: "snappy"
      restore:
        label: "Restore Configuration"
        components: {}
  config:
    openAPIV3Schema:
      type: object
      properties:
        type: { type: string, enum: [logical, physical], default: logical }
        compressionType:
          { type: string, enum: [none, gzip, snappy, lz4, s2, pgzip, zstd] }
        compressionLevel: { type: integer, minimum: 0, maximum: 22 }
  restoreConfig:
    openAPIV3Schema: { type: object, properties: {} }
```

### Provider: config consumption

Currently PSMDB provider's `SyncBackup()` ignores `Backup.spec.config`. Required changes in `provider-percona-server-mongodb`:

| Function            | Current                                             | Required                                                              |
| ------------------- | --------------------------------------------------- | --------------------------------------------------------------------- |
| `SyncBackup()`      | Sets only `ClusterName`, `StorageName`              | Read `backup.Spec.Config` → set `psmdbBackup.Spec.Type`, compression  |
| `SyncRestore()`     | Reads only PITR from `restore.Spec.DataSource.PITR` | Read `restore.Spec.Config` (future: selective restore params)         |
| `buildBackupSpec()` | PITR = simple bool from `storage.PITR.Enabled`      | Read `storage.PITR.Config` → set `oplogSpanMin`, compression for PITR |
| `BackupCustomSpec`  | Empty `struct{}`                                    | Not needed — config comes from Backup CR, not provider definition     |

---

## UI Component Architecture

### Component tree

```
Instance Details Page
└── Tab: Backups (backups.tsx)  — data source: API hooks, mutations: PATCH Instance
    │
    ├── <BackupsListTableHeader>
    │   ├── [Create Backup] ──→ <OnDemandBackupModal>
    │   │                        ├── Name (TextInput)                    STATIC
    │   │                        ├── BackupClass (SelectInput)           STATIC
    │   │                        ├── Storage (SelectInput)               STATIC
    │   │                        └── <UIGenerator section="backup" />    DYNAMIC
    │   │
    │   └── [Restore] ──→ <RestoreModal>
    │                      ├── Source info (read-only)                   STATIC
    │                      ├── PITR section (if supportsPITR)            STATIC
    │                      ├── <UIGenerator section="restore" />         DYNAMIC
    │                      └── ⚠️ Warning + [Restore] button
    │
    ├── Accordion: Active Storages
    │   │  One accordion section containing a list of storage cards.
    │   │  Each card = one entry in instance.spec.backup.storages[].
    │   ├── <StorageCard> per storage entry
    │   │   ├── Info: name, Main badge, PITR badge, schedules count
    │   │   ├── [Main] toggle (direct action on card)
    │   │   ├── [Configure PITR] ──→ <PITRConfigModal>
    │   │   │                        ├── PITR enabled toggle              STATIC
    │   │   │                        │   (disabled if maxWithPITR reached)
    │   │   │                        └── {pitrEnabled && <UIGenerator "pitr" />}  DYNAMIC
    │   │   ├── [+ Add Schedule] ──→ <ScheduledBackupModal>
    │   │   └── [Remove] ──→ <StorageRemoveConfirmDialog> (see "Storage removal")
    │   └── [+ Add Storage] → storage picker (see "Storage selection flow")
    │       (disabled if >= storageLimits.max)
    │
    ├── Accordion: Schedules ──→ <ScheduledBackupsList>
    │
    └── <BackupsList> (existing table)
        └── Row actions: [Restore] [Delete]
```

```
Instance Creation Wizard
└── Step: Backups  — data source: React Hook Form state, no API calls until submit
    │
    ├── [Enable Backups] toggle → ON
    ├── BackupClass select → loads storageLimits, supportsPITR, uiSchema
    │
    ├── Storage select (from available NS-level storages)
    │   └── [+ Create New Storage] → inline BackupStorage form
    │
    ├── Selected storage appears as <StorageCard>
    │   ├── [Main] toggle (auto: true for first)
    │   ├── [Configure PITR] → PITR toggle + <UIGenerator section="pitr" />
    │   └── [+ Add Schedule] → <ScheduleFormFields />
    │
    ├── [+ Add Another Storage] (if < storageLimits.max)
    │
    └── Submit → POST Instance with spec.backup assembled from form state
```

> **Wizard vs Backups tab — two orchestration modes:**
>
> | | Wizard (create/edit DB) | Backups tab (existing DB) |
> |---|---|---|
> | Instance | Does not exist yet | Exists |
> | Data source | React Hook Form state | API hooks (useQuery) |
> | Persistence | All at once on POST Instance | Each action → PATCH Instance |
> | Shared components | StorageCard (display), ScheduleFormFields, PITRConfigFields, UIGenerator | Same |
> | Orchestration | `<WizardBackupsStep />` | `<BackupsTab />` |
>
> **Presentational** sub-components are shared. **Orchestration** (where data comes from, how it's saved) is separate.

> **Future: Restore mode in wizard.** In v1/main, the DB wizard had a "restore from backup" mode. In v2 this will need UIGenerator `restore` section + backup/PITR source selection. Out of scope for initial implementation — requires separate design.

### Storage selection flow

The two-level storage model (NS-level BackupStorage CR + instance-level binding) is **hidden from the user**. From the user's perspective:

1. User clicks [+ Add Storage]
2. A picker shows available NS-level BackupStorages (filtered: those not yet bound to this instance)
3. User selects one (or creates new via [+ Create New Storage] → S3 credentials form)
4. The storage appears as a card — instance-level binding is created automatically
5. User configures PITR and schedules directly on the card

No separate "register storage on instance" step. No two forms for the same storage.

### Storage removal

Removing a storage from an instance = removing entry from `instance.spec.backup.storages[]`.

**Blocking conditions** (Remove button disabled with tooltip):
- Active backup in progress (Running/Pending) to this storage
- This is the `main` storage and there are other storages (must reassign main first)

**Warning conditions** (confirmation dialog lists consequences):
- Storage has N active schedules → "N schedules will be deleted"
- PITR is enabled on this storage → "PITR will be disabled"
- Storage has existing backups → "Existing backups remain accessible but no new backups to this storage"
- This is the only storage → "Backups will be fully disabled"

**Validation flow:**
1. UI pre-checks conditions from local state (schedules count, pitr.enabled, main flag, backup statuses)
2. Shows `<StorageRemoveConfirmDialog>` with consequence list
3. On confirm → PATCH Instance (remove from storages[])
4. BE validates server-side and can reject with error

### Static vs dynamic fields

| Context               | Static fields (React)          | Dynamic fields (UIGenerator)                                         |
| --------------------- | ------------------------------ | -------------------------------------------------------------------- |
| On-demand backup      | name, backupClass, storage     | `backup` section: e.g. backupType, compressionType, compressionLevel |
| PITR config           | pitr.enabled toggle            | `pitr` section: e.g. oplogSpanMin, compressionType                   |
| Restore               | source info, PITR date picker  | `restore` section: (empty for PSMDB initially)                       |
| Schedule              | name, cron, retention, enabled | _(none — fully static, maxSchedulesPerStorage as gating limit)_      |
| Storage card          | name, Main toggle              | _(none — display only + action buttons)_                             |

### UIGenerator integration pattern

```tsx
// Static React modal embeds UIGenerator for dynamic portion
const OnDemandBackupFieldsWrapper = () => {
  const selectedClassName = watch("backupClassName");
  const { data: backupClass } = useGetBackupClass(selectedClassName);
  const sections = backupClass?.spec?.providerManaged?.uiSchema;

  return (
    <>
      <TextInput name="name" label="Backup Name" />
      <SelectInput name="backupClassName" label="Backup Class" />
      <SelectInput name="storageName" label="Storage" />

      {/* Dynamic: rendered only if BackupClass provides a schema */}
      {sections?.backup && (
        <UIGenerator
          sectionKey="backup"
          sections={sections}
          formMode="new"
          namespace={namespace}
        />
      )}
    </>
  );
};
```

- UIGenerator renders inside the same `<FormProvider>` as static fields
- UIGenerator renders **only fields** — no section title, no description, no wrapper chrome
- No UIGenerator adaptation needed — it already works this way (used in `SectionEditModal`)
- On submit, dynamic values are packed into `spec.config`
- If no `uiSchema.backup` — no dynamic fields shown (graceful degradation)
- Select options (e.g. compressionType, backupType) are **static** values in `fieldParams.options`, not runtime data from an API. If dynamic options are needed in the future (e.g. compression depends on provider version), an `apiProvider` can be added per existing UIGenerator pattern

---

## User Flows

### Flow 1: Create DB with scheduled backups (wizard)

**Precondition:** At least one BackupStorage exists in the namespace (or user creates one inline).

```
User opens "Create Database" form
│
├── Steps 1-N: Instance config (topology, resources, etc.)
│
└── Step: Backups
    │
    ├── [Enable Backups] toggle → ON
    │
    ├── BackupClass select (auto-select if one)
    │   → loads storageLimits, supportsPITR, uiSchema
    │
    ├── Storage select (available NS-level BackupStorages)
    │   └── [+ Create New Storage] → inline S3 form
    │       → creates BackupStorage CR in namespace
    │       → auto-selects it
    │
    ├── Selected storage appears as <StorageCard>
    │   ├── Main: true (auto for first)
    │   ├── [Configure PITR] (if supportsPITR)
    │   │   ├── PITR toggle → ON
    │   │   └── <UIGenerator section="pitr" />
    │   │       └── Oplog Span: 10 min, Compression: snappy
    │   └── [+ Add Schedule] → <ScheduleFormFields />
    │       ├── Name: "daily-full"
    │       ├── Cron: Every day at 2:00 AM
    │       ├── Retention: 7 copies
    │       └── [Save] → schedule appears on card
    │
    ├── [+ Add Another Storage] (if < storageLimits.max)
    │
    └── [Submit] → POST Instance with spec.backup:
        enabled: true
        classRef: { name: "psmdb-managed" }
        storages:
        - name: "s3-main"
          storageRef: { name: "my-s3-storage" }
          main: true
          pitr: { enabled: true, config: { oplogSpanMin: 10 } }
          schedules:
          - { name: "daily-full", cron: "0 2 * * *",
              retentionCopies: 7, enabled: true }
```

> **Note:** All data is collected in React form state. No PATCH requests during wizard — everything is sent as part of POST Instance on submit. The instance-level binding (`spec.backup.storages[]`) is assembled from form state automatically — user never sees the concept of "registering a storage on an instance".

### Flow 2: Create on-demand backup

```
Instance Details → Tab: Backups → [Create Backup]
│
▼
<OnDemandBackupModal>
├── Backup Name (auto-generated, editable, unique check)
├── BackupClass select → fetches uiSchema.backup
├── Storage select (registered + unregistered namespace storages)
├── <UIGenerator section="backup" />  ← dynamic fields
│   ├── Backup Type: [Logical ▼]
│   ├── Compression: [snappy ▼]
│   └── Compression Level: [6]
└── [Create]
    ├── If unregistered storage → PATCH Instance to add storage
    └── POST Backup CR with spec.config from UIGenerator
```

### Flow 3: Create schedule — no storages configured yet

```
Instance Details → Tab: Backups
│
├── storages = [] → "No storages configured"
│   └── [+ Add Storage] → storage picker
│       ├── Select from available NS-level BackupStorages
│       ├── [+ Create New Storage] → S3 form → creates BackupStorage CR
│       └── [Select] → PATCH Instance (add to storages[], auto-enable backups)
│
├── Storage card appears (main: true, auto)
│   └── [+ Add Schedule] → <ScheduledBackupModal>
│       ├── Storage: pre-selected
│       ├── Name, Cron, Retention, Enabled
│       └── [Create] → PATCH Instance (conflict retry)
│
└── Schedule appears in Schedules accordion
```

### Flow 4: Create schedule — storage already exists

```
Instance Details → Tab: Backups
│
├── Active Storages: <StorageCard name="s3-prod"> (Main, 0 schedules)
│
├── [+ Add Schedule] on card → <ScheduledBackupModal>
│   ├── Storage: "s3-prod" (pre-filled)
│   ├── Schedule Name: "daily-backup"
│   ├── Cron: Every day at 2:00 AM
│   ├── Retention: 7 copies
│   ├── Enabled: ON
│   └── [Create] → PATCH Instance (conflict retry)
│
└── Schedules table:
    | Name         | Storage | Schedule        | Retention | Status  | Actions      |
    | daily-backup | s3-prod | Every day, 2 AM | 7 copies  | Enabled | [Edit] [Del] |
```

### Flow 5: Restore from backup

```
Instance Details → Tab: Backups
│
├── Backups list: backup-001 (Succeeded, 2.1GB)
│
├── [Restore] on backup-001 → <RestoreModal>
│   ├── Source info (read-only): backup name, storage, date, size
│   ├── PITR section (if storage had PITR enabled):
│   │   ├── ○ Restore to backup point (default)
│   │   └── ○ Restore to point in time → Date picker
│   ├── <UIGenerator section="restore" />  ← dynamic (empty for PSMDB initially)
│   ├── ⚠️ "Restoring will replace all data. Cannot be undone."
│   └── [Restore]
│       → POST Restore CR with dataSource.pitr + config
│
├── Instance: Ready → Restoring → Ready
│
└── Restores tab: restore-xyz | backup-001 | Succeeded
```

---

## Mock Data

### Mock BackupClass with uiSchema

```typescript
// ui/apps/everest/src/mocks/backup-class-mock.ts

import { Section } from "../components/ui-generator/ui-generator.types";

export const psmdbBackupClassUISchema: Record<string, Section> = {
  backup: {
    label: "Backup Configuration",
    componentsOrder: ["type", "compressionType", "compressionLevel"],
    components: {
      type: {
        uiType: "select",
        label: "Backup Type",
        path: "type",
        required: true,
        fieldParams: {
          options: [
            { label: "Logical", value: "logical" },
            { label: "Physical", value: "physical" },
          ],
          defaultValue: "logical",
        },
      },
      compressionType: {
        uiType: "select",
        label: "Compression",
        path: "compressionType",
        fieldParams: {
          options: [
            { label: "None", value: "none" },
            { label: "Gzip", value: "gzip" },
            { label: "Snappy", value: "snappy" },
            { label: "LZ4", value: "lz4" },
            { label: "Zstandard", value: "zstd" },
          ],
          defaultValue: "snappy",
        },
      },
      compressionLevel: {
        uiType: "number",
        label: "Compression Level",
        path: "compressionLevel",
        fieldParams: { defaultValue: 6 },
        validation: { minimum: 0, maximum: 22 },
      },
    },
  },
  pitr: {
    label: "PITR Configuration",
    componentsOrder: ["oplogSpanMin", "compressionType"],
    components: {
      oplogSpanMin: {
        uiType: "number",
        label: "Oplog Span (minutes)",
        path: "oplogSpanMin",
        tooltip: "Interval between oplog chunk boundaries",
        fieldParams: { defaultValue: 10 },
        validation: { minimum: 1 },
      },
      compressionType: {
        uiType: "select",
        label: "Oplog Compression",
        path: "compressionType",
        fieldParams: {
          options: [
            { label: "None", value: "none" },
            { label: "Snappy", value: "snappy" },
            { label: "Zstandard", value: "zstd" },
          ],
          defaultValue: "snappy",
        },
      },
    },
  },
  restore: {
    label: "Restore Configuration",
    components: {},
  },
};
```

### Mock hook strategy

```typescript
// ui/apps/everest/src/hooks/api/backup-classes/useBackupClassWithUISchema.ts

import { useGetBackupClass } from "./useBackupClasses";
import { psmdbBackupClassUISchema } from "../../../mocks/backup-class-mock";

// Temporary: wraps real hook and injects mock uiSchema.
// Remove when BE populates providerManaged.uiSchema.
export const useBackupClassWithUISchema = (className?: string) => {
  const query = useGetBackupClass(className);
  return {
    ...query,
    data: query.data
      ? {
          ...query.data,
          spec: {
            ...query.data.spec,
            providerManaged: {
              ...query.data.spec?.providerManaged,
              uiSchema: psmdbBackupClassUISchema,
            },
          },
        }
      : undefined,
  };
};
```

---

## Implementation Phases

### Phase 0: Backend — CRD extension + provider support

| #   | Task                                                                        | File / Repo                                                              | Depends on |
| --- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------ | ---------- |
| 0a  | Extend `ProviderManagedSpec` with `UISchema`, `StorageLimits`, `PITRConfig` | `api/backup/v1alpha1/backupclass_types.go`                               | —          |
| 0b  | Regenerate CRD OpenAPI + client                                             | `make generate && make generate-client`                                  | 0a         |
| 0c  | Create BackupClass manifest for PSMDB                                       | `provider-percona-server-mongodb/charts/.../templates/backup-class.yaml` | 0a         |
| 0d  | Read `Backup.spec.config` in `SyncBackup()` → set backup type + compression | `provider-percona-server-mongodb/internal/provider/backup.go`            | 0a         |
| 0e  | Read `pitr.config` in `buildBackupSpec()` → set oplogSpanMin, compression   | same file                                                                | 0a         |

> **Note:** UI work (Phases 1-7) can proceed in parallel using mock data. Phases 0 and 1+ are independent tracks.

### Phase 1: Foundation — mocks + UISchema support

| #   | Task                                           | File                                                            | Depends on |
| --- | ---------------------------------------------- | --------------------------------------------------------------- | ---------- |
| 1   | Create mock BackupClass with uiSchema sections | `ui/.../mocks/backup-class-mock.ts`                             | —          |
| 2   | Create mock hook `useBackupClassWithUISchema`  | `ui/.../hooks/api/backup-classes/useBackupClassWithUISchema.ts` | 1          |
| 3   | Add `getBackupClassUISection()` utility        | `ui/.../utils/backup-class-utils.ts`                            | —          |
| 4   | Test UIGenerator integration with mock section | test or story                                                   | 1, 2, 3    |

### Phase 2: On-demand backup modal enhancement

| #   | Task                                                            | File                                                                | Depends on |
| --- | --------------------------------------------------------------- | ------------------------------------------------------------------- | ---------- |
| 5   | Add UIGenerator to `OnDemandBackupFieldsWrapper`                | `ui/.../on-demand-backup-modal/on-demand-backup-fields-wrapper.tsx` | 3          |
| 6   | Update submit handler to pack dynamic fields into `spec.config` | `ui/.../on-demand-backup-modal/on-demand-backup-modal.tsx`          | 5          |
| 7   | Update Zod schema for dynamic config                            | `ui/.../on-demand-backup-modal/on-demand-backup-modal.types.ts`     | 6          |

### Phase 3: Storage management (Backups tab)

| #   | Task                                                                  | File                                                               | Depends on |
| --- | --------------------------------------------------------------------- | ------------------------------------------------------------------ | ---------- |
| 8   | Migrate BackupStorage hooks to v2 API                                 | `ui/.../hooks/api/backup-storages/`, `ui/.../api/backupStorage.ts` | —          |
| 9   | Create `<StorageCard />` (presentational — shared between tab & wizard) | `ui/.../backups/storage-card/storage-card.tsx`                     | 8          |
| 10  | Create `<PITRConfigModal />` with toggle + UIGenerator `pitr` section | `ui/.../backups/pitr-config-modal/pitr-config-modal.tsx`           | 3, 9       |
| 11  | Create `<StorageRemoveConfirmDialog />`                                | `ui/.../backups/storage-remove-confirm-dialog.tsx`                 | —          |
| 12  | Add Active Storages accordion + storage picker to Backups tab         | `ui/.../backups/backups.tsx`                                       | 9, 10, 11  |

### Phase 4: Scheduled backups

| #   | Task                                   | File                                                                     | Depends on |
| --- | -------------------------------------- | ------------------------------------------------------------------------ | ---------- |
| 13  | Create `<ScheduleFormFields />`        | `ui/.../backups/scheduled-backup-modal/schedule-form-fields.tsx`         | —          |
| 14  | Implement `<ScheduledBackupModal />`   | `ui/.../backups/scheduled-backup-modal/scheduled-backup-modal.tsx`       | 13         |
| 15  | Implement `<ScheduledBackupsList />`   | `ui/.../backups/backups-list/table-header/scheduled-backups-list.tsx`    | 14         |
| 16  | Wire schedule button on storage card   | `ui/.../backups/storage-card/storage-card.tsx`                           | 14, 12     |

### Phase 5: Restore

| #   | Task                                     | File                                               | Depends on |
| --- | ---------------------------------------- | -------------------------------------------------- | ---------- |
| 17  | Add `createRestoreFn`                    | `ui/.../api/restores.ts`                           | —          |
| 18  | Create `useCreateRestore` hook           | `ui/.../hooks/api/restores/useDbClusterRestore.ts` | 17         |
| 19  | Create `<RestoreModal />`                | `ui/.../backups/restore-modal/restore-modal.tsx`   | 3, 18      |
| 20  | Add restore button to backup row actions | `ui/.../backups/backups-list/backups-list.tsx`     | 19         |

### Phase 6: Instance creation wizard — backups step

Wizard step has **separate orchestration** from the Backups tab: it reads/writes React Hook Form state, not API.
Shared presentational components: `StorageCard`, `ScheduleFormFields`, `PITRConfigFields` (UIGenerator wrapper).

| #   | Task                                               | File                                             | Depends on |
| --- | -------------------------------------------------- | ------------------------------------------------ | ---------- |
| 21  | Create `<WizardBackupsStep />` with form-state logic | `ui/.../database-form/steps/backups-step/`       | 9, 13      |
| 22  | Register `backupClasses` api-provider              | `ui/.../ui-generator/api-providers/providers.ts` | —          |
| 23  | Register `backupStorages` api-provider             | `ui/.../ui-generator/api-providers/providers.ts` | —          |

### Phase 7: Cleanup

| #   | Task                                  | File                 | Depends on |
| --- | ------------------------------------- | -------------------- | ---------- |
| 24  | Remove legacy v1 backup types/imports | various              | all above  |
| 25  | Remove commented-out PITR hook code   | `useBackups.ts`      | all above  |
| 26  | Remove old v1 backup form step        | `steps-old/backups/` | 21         |
| 27  | Enable RBAC checks in hooks           | various hooks        | all above  |

---

## File Inventory

### Files to create (UI)

| File                                                                         | Purpose                                                       |
| ---------------------------------------------------------------------------- | ------------------------------------------------------------- |
| `ui/apps/everest/src/mocks/backup-class-mock.ts`                             | Mock BackupClass with providerManaged.uiSchema                |
| `ui/apps/everest/src/hooks/api/backup-classes/useBackupClassWithUISchema.ts` | Mock hook (temporary)                                         |
| `ui/apps/everest/src/utils/backup-class-utils.ts`                            | `getBackupClassUISection()` utility                           |
| `ui/.../backups/storage-card/storage-card.tsx`                                | Presentational storage card (shared: tab + wizard)            |
| `ui/.../backups/pitr-config-modal/pitr-config-modal.tsx`                      | PITR toggle + UIGenerator `pitr` section                      |
| `ui/.../backups/storage-remove-confirm-dialog.tsx`                            | Confirmation dialog with pre-check warnings                   |
| `ui/.../backups/scheduled-backup-modal/schedule-form-fields.tsx`              | Presentational schedule form fields (shared: tab + wizard)    |
| `ui/.../backups/restore-modal/restore-modal.tsx`                              | Restore creation modal                                        |
| `ui/.../database-form/steps/backups-step/backups-step.tsx`                    | Wizard backups step (form-state orchestration)                |

### Files to modify (UI)

| File                                                                     | Change                                    |
| ------------------------------------------------------------------------ | ----------------------------------------- |
| `ui/.../on-demand-backup-modal/on-demand-backup-fields-wrapper.tsx`      | Add UIGenerator `backup` section          |
| `ui/.../on-demand-backup-modal/on-demand-backup-modal.tsx`               | Pack dynamic fields into spec.config      |
| `ui/.../on-demand-backup-modal/on-demand-backup-modal.types.ts`          | Extend Zod schema                         |
| `ui/.../backups/backups.tsx`                                             | Add Active Storages accordion + storage picker + Schedules accordion |
| `ui/.../backups/backups-list/table-header/backups-list-table-header.tsx` | Wire schedule + restore buttons           |
| `ui/.../backups/backups-list/backups-list.tsx`                           | Add restore row action                    |
| `ui/.../backups/scheduled-backup-modal/scheduled-backup-modal.tsx`       | Un-stub, implement                        |
| `ui/.../backups/backups-list/table-header/scheduled-backups-list.tsx`    | Un-stub, implement                        |
| `ui/.../hooks/api/backup-storages/useBackupStorages.ts`                  | Migrate to v2 API                         |
| `ui/.../api/backupStorage.ts`                                            | Migrate endpoints to v2                   |
| `ui/.../api/restores.ts`                                                 | Add createRestoreFn                       |
| `ui/.../hooks/api/restores/useDbClusterRestore.ts`                       | Add useCreateRestore                      |
| `ui/.../ui-generator/api-providers/providers.ts`                         | Register backupClasses + backupStorages   |

### Files to modify (BE)

| File / Repo                                                              | Change                       |
| ------------------------------------------------------------------------ | ---------------------------- |
| `api/backup/v1alpha1/backupclass_types.go`                               | Extend `ProviderManagedSpec` |
| `api/openapi/crds.gen.yaml`                                              | Regenerated                  |
| `client/everest-client.gen.go`                                           | Regenerated                  |
| `provider-percona-server-mongodb/charts/.../templates/backup-class.yaml` | New: BackupClass manifest    |
| `provider-percona-server-mongodb/internal/provider/backup.go`            | Read config/pitr.config      |

---

## Verification Criteria

1. On-demand modal renders dynamic fields from BackupClass → config sent in `Backup.spec.config`
2. Active Storages accordion shows instance storages; edit modal has PITR toggle + dynamic config
3. Scheduled backups CRUD per storage via Instance PATCH
4. Restore modal shows source info, PITR date picker, dynamic restore config
5. Instance wizard backups step collects spec.backup from form state
6. All forms degrade gracefully when BackupClass has no `uiSchema`
7. `pnpm lint` and `pnpm typecheck` pass
8. Existing on-demand backup + backups list + restores list unchanged
9. BE: `ProviderManagedSpec` extended, CRD/client regenerated
10. BE: PSMDB provider creates BackupClass at install, reads `Backup.spec.config` in `SyncBackup()`
