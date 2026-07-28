# PITR — Test Cases

Only the code that already exists: `pitr.utils`, the PITR toggle, and the storages panel.
Cases for features that aren't built yet (config modal, restore-from-PITR) are intentionally
left out until that code exists — otherwise they'd drift when the design changes.

Marker: ✅ written · ⬜ todo.

## Unit — `pitr.utils`

1. ✅ `setStoragePitr` — if the name matches a storage, only that storage's `pitr` changes; the others keep their reference.
2. ✅ `setStoragePitr` — if no storage matches the name, the list is returned unchanged.
3. ✅ `countPitrEnabledStorages` — returns how many storages have `pitr.enabled`.
4. ✅ `hasActiveSchedules` — true only when some storage has an enabled schedule.

## Component — PITR toggle

1. ✅ if the provider doesn't support PITR → no toggle is rendered.
2. ✅ if PITR is off and the user turns it on → PATCH sets `storages[i].pitr.enabled = true`.
3. ✅ if no storage has an active schedule → toggle disabled, "needs schedule" tooltip.
4. ✅ if the PITR-enabled limit is reached → toggle disabled, "limit reached" tooltip.
5. ⬜ if PITR is on and the user turns it off → PATCH sets `enabled = false`, existing `parameters` preserved.
6. ⬜ if both "no schedule" and "limit reached" apply → the "needs schedule" tooltip wins.

## Component — storages panel

1. ✅ expanding the storages toggle → one row per `spec.backup.storages[]`.
2. ⬜ opening one panel (schedules or storages) closes the other → only one open at a time.

## E2E — smoke (real PSMDB)

1. ⬜ enabling PITR on a storage row of a running PSMDB instance → `pitr.enabled = true` persists in the CRD.
