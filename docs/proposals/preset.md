
# Presets

## Overview

A **Preset** is provider-specific default configuration values. Presets replace the current ad-hoc configuration patterns (split horizon config, load balancer annotations, pod scheduling policy) with a unified mechanism.

Presets are scoped to a single provider and cannot be shared across providers.

## Goals

1. ✅ **One-click deployment** — deploy an Instance with sensible defaults without manually configuring each component. *(Phase 1)*
2. ✅ **Custom values** — users use preset as starting point and override configurations during Instance creation. *(Phase X)*
3. ✅ **Bulk update** — when a preset is updated, all Instances using its defaults (without user overrides) are updated automatically. *(Phase Y)*

## User Requirements

### 1. One-Click Deployment

- As a user, I can deploy an Instance with one click on the Everest UI, and the system applies a preset automatically so I don't need to configure each component manually.
- As a user, I can install OpenEverest with preset already configured.
- As a user, I can customize preset and create new preset in OpenEverest UI (not initial phase)
- As a user, I can select which preset to use for one-click deployment when multiple presets exist.

### 2. Custom Values (not initial phase)

- As a user, I can see available presets during Instance creation and select one as a starting point.
- As a user, I can override preset if admin allows override.
- As a user, I can override individual values from the selected preset during Instance creation (e.g. instead of creating another configuration)
- As a user, I can create a preset in OpenEverest UI.
- As a user, I can create a preset in OpenEverest UI based on a running Instance. (Create Preset action in the Instance)
- As an admin, I can control which presets are visible to which users based on their role (e.g., `lb-prod` is visible to the platform team but not to the product team).

### 3. Bulk Update (separate feature)

- As an admin, I can update a preset and all Instances using that preset's defaults are updated or not updated based on choice.
  - update via Everest UI
  - update via `kubectl`
  - update via Helm
- (undecided) As a user, my explicit overrides are preserved when a preset is updated — only fields I did not override receive the new defaults.

## Plans

### 🚀 Phase 1 (Initial Release)

- ✅ One-click deployment with preset applied automatically
- ✅ Install OpenEverest with preset already configured
- ✅ Select which preset to use when multiple presets exist

### 🔮 Phase X (Future)

- 🔜 Customize and create new presets in the UI
- 🔜 Override individual preset values during Instance creation
- 🔜 Admin controls preset visibility per user/role
- 🔜 Create a preset based on a running Instance

### 📦 Phase Y (Bulk Update)

- 🔜 Update a preset and propagate changes to Instances
- 🔜 Preserve user overrides when a preset is updated

## Phase 1

### Goals

1. Introduce the **Preset** Custom Resource (CR) to define provider-specific default configurations.
2. UI reads values from the Preset CR and pre-fills the Instance creation form.
3. One-click deployment — user selects a preset and deploys an Instance without manual configuration.

### Preset CR

- A Preset CR is namespace scoped resource, it sets namespaced aware resources such as BackupStorage.
- A Preset CR contains the same configuration fields as an Instance CR (e.g., replicas, storage, resources, backup, proxy, load balancer annotations, scheduling, etc.).
- A Preset CR is installed with OpenEverest.

### OpenEverest UI

- When creating an Instance, the UI fetches available Preset CRs for the selected provider.
- If multiple Presets exist, the user can select which preset to apply.
- Preset values populate instance and the user submits to create the Instance.

### Validation

- The pre-installed Preset CR ships with known-valid values.
- A Preset CR follows the **same validation rules** as an Instance CR; runtime validation will be added in a future phase when users can create/edit presets via the UI.
