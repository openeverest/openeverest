# UI Testing Architecture — Roadmap & Ideas

Status: **draft for team discussion**. Last updated: 2026-08-06.

This document captures the medium/long-term plan for the OpenEverest **UI**
test architecture. Use this file to turn each roadmap item into a GitHub issue
after team review.

---

## 1. Guiding principles

1. **Test pyramid.** Most coverage lives in fast unit/component tests; a thin
   layer of live integration (e2e) proves the whole thing actually works. Live
   e2e is expensive (real k8s + Go backend + provider) — spend it only on
   golden paths.
2. **Separate _what_ from _where_.** The same behavior can be verified at
   different layers. Decide the cheapest layer that still gives a trustworthy
   signal.
3. **Mocks drift (mock drift).** Unit tests with hand-written mocks pin the FE
   to _its own assumptions_, not to reality. When the backend changes behavior
   (not just types), mocks stay green while production breaks. We must have at
   least one layer that runs the real UI against the real API.
4. **A matrix is not integration confidence.** Running `engines × sizes × flags`
   in e2e does not increase drift detection — every branch hits the same
   endpoints/types. One golden path catches API drift just as well, far cheaper.
   Keep the matrix in unit/component tests; keep e2e to golden paths.

### Three levels of drift (and who catches each)

| Level    | What drifts                                                              | Guard                                  |
| -------- | ------------------------------------------------------------------------ | -------------------------------------- |
| Types    | regenerated types don't compile                                          | `tsc` on regen (already in place)      |
| Contract | types compile, but payload shape/enum/optionality diverges from the mock | **fixtures-vs-OpenAPI guard** (item 3) |
| Behavior | types + contract OK, but the UI breaks on live data                      | **live golden e2e** (item 4)           |

---

## 2. Test layers in this repo

| Layer                    | Tool / location                             | Runs where                                               | Needs cluster?     |
| ------------------------ | ------------------------------------------- | -------------------------------------------------------- | ------------------ |
| Unit / component         | Vitest, `ui/apps/everest/src/**/*.test.tsx` | `dev-fe-gatekeeper` on every `ui/**` PR                  | no                 |
| Component (real browser) | Vitest browser mode, `*.browser.test.tsx`   | `dev-fe-browser-tests` (workflow_call)                   | no (real Chromium) |
| API contract             | Playwright, `api-tests/tests/*.spec.ts`     | `dev-be-ci` → `integration_tests_api` (k3d)              | yes, no UI         |
| Live UI e2e              | Playwright, `ui/apps/everest/.e2e/**`       | `dev-fe-e2e` (workflow_call, minikube + real Go backend) | yes                |

Note: `dev-fe-e2e` builds the Everest API server + controller, installs via
`everestctl`, and installs DB engines/providers **through the OLM catalog**
(`--operator.mongodb/postgresql/mysql=true`, `olm.catalogSourceImage`), not via
Tilt (Tilt is local-dev only).

---

## 3. Decision: per-PR live e2e uses ONE reference provider (PXC)

To keep PR CI fast, the live UI e2e lane runs against **a single reference
provider: PXC (percona-xtradb-cluster)**.

Rationale:

- PXC ships the richest `uiSchema` — including provider-specific parameter
  fields and their config modal — so it exercises the most UI surface per run.
- Other engines differ from the UI's point of view mostly by **schema/flags**,
  which is deterministic UI logic best covered by **unit/component tests +
  curated fixtures** (see item 2 below), not by multiplying slow live runs.

Consequence: engine/flag variations are covered by mocks; their fidelity is
protected by the fixture sync-check (item 1) and the fixtures-vs-OpenAPI guard
(item 3).

---

## 4. Roadmap items

Each item is written so it can become a standalone issue.

### Item 1 — Curated provider `uiSchema` fixtures + sync-check

> **TODO (revisit later).** Provider `uiSchema` mocks in our tests are
> hand-written copies of the real provider `ui.yaml`, so they can silently drift
> from the real schemas (which live in the provider repos and, at runtime, in
> `BackupClass.spec.uiSchema`). We need a curated fixture set plus a sync-check
> that flags when the vendored copies diverge from upstream — this keeps our
> mock-based, multi-provider coverage trustworthy without running every provider
> live. To work through: pick the source of truth (provider-repo file vs live
> `BackupClass` vs vendored copy) and the cross-repo sync mechanism, and settle
> that in-team before delegating the mechanical fixture work.

### Item 2 — Adversarial UIGenerator robustness tests

**Problem.** A provider author can write a malformed/unusual `uiSchema` and only
find out when connecting to core + UI. The UI must degrade gracefully, never
crash.

**Sketch.** Component-level tests that feed **pathological** schemas into
UIGenerator: empty schema, unknown `uiType`, missing `label`, deep nesting,
malformed `validation`, weird `path`. Assert: never throws, unknown field types
are skipped/fallback-rendered, the form still submits. This is generic (no real
provider schemas needed) and cheap — highest leverage for "UI doesn't crash on
arbitrary yaml".

**Effort.** Small. Sits next to existing `components/ui-generator/unit-tests/`.

**Delegation.** Good community candidate once a couple of examples exist.

### Item 3 — Fixtures-vs-OpenAPI contract guard

**Problem.** Hand-written mocks are assumed to match the API contract, but
nothing verifies it. On type regen a mock can silently diverge.

**Two levels:**

- **(a) Compile-time (cheap, do first).** Type each mock fixture with the
  generated OpenAPI types so `tsc` fails on drift. Example shape:

  ```ts
  import type { paths } from "api/http-api.types";
  type ResourceListResponse =
    paths["/.../<resource>"]["get"]["responses"]["200"]["content"]["application/json"];
  export const resourceListMock: ResourceListResponse = {
    /* ... */
  };
  ```

  Now regenerating the types (`make generate-openapi-types`) that changes the
  contract makes the fixture fail to compile — a free contract guard on the same
  PR, no runtime, no cluster.

- **(b) Runtime (for recorded real responses).** For fixtures captured from a
  live API, validate them against the generated schema with a JSON-schema
  validator (e.g. `ajv`) so we catch cases where the runtime payload violates
  what the types _claim_ (e.g. a "required" field actually missing).

**Effort.** (a) small and immediate; (b) medium, needs a recording step.

**Delegation.** (a) in-team (touches the generated-types setup); (b) later.

### Item 4 — BE→FE gate (backend gated by frontend reality before merge)

**Problem.** `dev-be-ci.yaml` has `paths-ignore: ['ui/**']` and the FE gatekeeper
runs on `ui/**` — they are mutually exclusive. So a backend PR (`api/**`,
`internal/server/**`) runs BE CI but **does not** run the live FE e2e. A backend
behavior change can merge while silently breaking the UI.

**Sketch.** `dev-fe-e2e.yaml` is already a reusable `workflow_call`. Add a job in
`dev-be-ci.yaml` that calls it (mirror the commented `cli-tests.yml` pattern):

```yaml
fe_e2e_against_be:
  uses: ./.github/workflows/dev-fe-e2e.yaml
  secrets: inherit
```

Ideally gate it to contract-relevant paths (`api/**`, `internal/server/**`) so
we don't run the heavy lane on trivial backend changes. Add it to the
`merge-gatekeeper` `needs` list so it blocks merge.

**Effort.** Small (CI wiring). Cost: adds a heavy job to backend PRs — scope by
path.

**Delegation.** In-team (CI + release engineering).

### Item 5 — Nightly scheduled golden run

**Problem.** Per-PR golden paths only catch drift caused by a change **in this
repo**. External moving parts drift out-of-band: `dev-fe-e2e` pulls floating
`operator-dev` / `catalog-dev` images, the PMM chart, and the version service.
A provider/operator can change under us with no OpenEverest PR.

**Sketch.** Run the golden paths on a schedule (nightly) to catch out-of-band
semantic drift. Same tests, cron trigger.

**Note.** If all external images were pinned, nightly would be redundant. While
`-dev` tags float, it is worth it.

**Effort.** Small.

### Item 6 — Shared UI-render conformance (bidirectional contract)

**Goal.** Let provider authors know **in their own CI** when their `ui.yaml`
breaks the real UI, and let core changes signal to providers when rendering
behavior breaks.

**The hard constraint.** UIGenerator depends on core hooks/context. We do **not**
want to extract all of core, and extracting only UIGenerator is problematic
because of that hook coupling.

**Practical shape — a versioned "schema validator" artifact, not a library.**

- Build a small **headless validator tool/container** inside openeverest that
  bundles the full UI build, **mocks the core hooks/API**, takes a `ui.yaml`
  (or a live `BackupClass`) as input, renders the relevant sections through the
  real UIGenerator, and reports pass/fail (+ optional static meta-schema
  validation). Providers do **not** import UIGenerator source — they run the
  published container in their CI against their own schema.
- **Version it (semver / tagged image).** Providers pin a validator version.
  - **Producer → Consumer:** a provider's CI runs the validator on its `ui.yaml`
    → catches a bad schema before integration.
  - **Consumer → Producer:** when core makes a breaking rendering change, bump
    the validator's major. Providers pinned to the old major see they must
    adapt; providers tracking the floating tag catch it in nightly against the
    fresh core image.
- Two modes in one tool: (1) static validation against the published **meta-
  schema** (cheap, no render), (2) actual headless render (catches real
  breakage).

**Effort.** Large. This is the biggest item — build + maintain the validator
container and its mock layer. Future work, not now.

**Open design questions for the team.**

- Container vs a thin published npm harness that still needs a host app?
- What exactly do we mock for the core hooks (queries, context providers)?
- Does the meta-schema live in the Provider SDK (`provider-runtime`) so all
  providers share it?

### Item 7 — Unit test sharding (when it hurts)

Currently `make test` runs unit + browser tests on a **single runner** — no
sharding. When the suite gets slow, add `vitest run --shard=i/n` with a GitHub
Actions `strategy.matrix` over shard indices. Not needed at current volume;
noted so we know the lever exists.

---

## 5. Open questions for the team

- Source of truth for provider `uiSchema` fixtures: provider repo file, live
  `BackupClass`, or vendored copy + sync-check? (item 1)
- Do we build the validator container (item 6), or is per-PR reference-provider
  e2e + adversarial unit tests (items 2–3) "good enough" for now?
- Where does the shared **meta-schema** live — Provider SDK (`provider-runtime`)?
- BE→FE gate path scoping: which backend paths should trigger the heavy FE lane?
