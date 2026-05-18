# LFX Mentorship 2026 Term 2 Proposal

## CNCF / OpenEverest  
### Plugin Developer Playground: Interactive UI Schema Editor with Live Preview (V2)

---

# Preliminary Questions

---

## 1. How did you find out about the mentorship program?

I regularly track both the CNCF Mentoring repository and the LFX Mentorship dashboard each term to look for projects aligned with frontend infrastructure, developer tooling, and OSS platform engineering.

Issue `openeverest/openeverest#2059` immediately stood out because it combines several areas I actively work in:

- React + TypeScript application architecture
- schema-driven UI systems
- browser security and CSP constraints
- developer tooling workflows

Before deciding to apply, I reviewed:

- the upstream issue discussion
- the V2 plugin architecture on the `v2` branch
- the existing `/ui-generator-builder` proof-of-concept
- the UIGenerator runtime flow
- the current CSP implementation

That initial investigation confirmed that the project was both technically meaningful and realistically implementable within the mentorship timeline.

---

## 2. Why are you interested in this program?

OpenEverest V2 introduces a powerful plugin architecture where database providers define their UIs declaratively through YAML schemas instead of shipping custom React implementations.

That architecture becomes significantly more valuable when plugin authors can iterate quickly.

Right now, the development workflow is slow and infrastructure-heavy because authors must:

1. update a Provider CRD
2. apply it to Kubernetes
3. reload the UI
4. navigate back to the correct form
5. repeat the cycle for every schema change

The existing playground proof-of-concept attempted to solve this problem but currently depends on Monaco Editor, which violates OpenEverest’s strict production CSP because of runtime `new Function()` usage.

This proposal focuses on building a production-safe replacement using:

- CodeMirror 6
- runtime UIGenerator reuse
- inline validation
- mock provider support
- CSP-safe editor integration

What especially interested me is that the challenge is not just “build an editor.”

It sits at the intersection of:

- frontend architecture
- runtime systems
- security engineering
- developer experience
- OSS maintainability

which is exactly the kind of engineering work I want to keep pursuing long term.

---

### Additional Motivation

During proposal preparation, I also implemented and validated a local CSP-safe proof-of-concept directly against the V2 branch.

That work included:

- replacing Monaco with CodeMirror 6
- propagating the existing CSP nonce flow
- validating editor behavior under production CSP
- integrating inline YAML diagnostics
- wiring route-level playground infrastructure
- building a split-pane editor/preview workflow
- adding initial unit/component test coverage

This early validation significantly reduced architectural uncertainty and confirmed that the proposed direction works within OpenEverest’s existing security model without requiring backend CSP changes.

---

## 3. What experience and knowledge/skills do you have that are applicable to this program?

My primary experience is in:

- React
- TypeScript
- frontend architecture
- developer tooling
- validation-heavy UI systems

I regularly work with:

- component-driven frontend systems
- structured state management
- modular TypeScript architecture
- schema-oriented workflows
- test-oriented development

---

### Relevant Project Experience

#### Electron Productivity Application

Built using:

- Electron
- React
- TypeScript

Key areas involved:

- modular frontend architecture
- desktop runtime constraints
- application state management
- long-term maintainability
- production-focused UX design

---

#### PR Identity Verification System

A workflow-focused system centered around:

- automated validation
- structured processing pipelines
- frontend/backend coordination
- reliability-focused architecture

This project especially strengthened my understanding of:

- validation flows
- edge-case handling
- structured runtime processing
- implementation scalability

---

### OpenEverest-Specific Preparation

For this proposal, I spent significant time studying the OpenEverest V2 plugin system directly from the repository instead of relying only on the issue description.

Repository areas studied include:

#### UIGenerator Runtime

- `ui-generator.tsx`
- `ui-generator.types.ts`
- `use-form-engine.ts`
- `use-topology.ts`

#### Provider Registry System

- `api-providers/registry.ts`
- `api-providers/types.ts`

#### Schema Processing Pipeline

- `preprocess-schema.ts`
- `postprocess-schema.ts`

#### Existing Playground POC

- `pages/ui-generator-builder/`

#### CSP & Frontend Security Flow

- `internal/server/middlewares.go`
- `App.tsx`
- `index.html`

---

### Existing Proof-of-Concept Work

As part of proposal preparation, I also implemented a local CSP-safe playground proof-of-concept directly against the V2 frontend architecture.

The PoC validated:

- successful CodeMirror 6 integration
- CSP nonce propagation
- YAML diagnostics
- split-pane rendering
- runtime UIGenerator compatibility
- route-level integration
- production-build compatibility

The prototype also included:

- accessibility-aware keyboard handling
- validation plumbing
- initial test coverage
- zero CSP violations under production preview builds

This preparation helped convert several architectural unknowns into validated implementation paths before the mentorship period even begins.

---

## 4. What do you hope to get out of this mentorship experience?

There are three major things I hope to gain from this mentorship.

---

### 1. Production-Quality Frontend Engineering Experience

I want deeper experience building and maintaining production-grade frontend systems inside a large OSS codebase.

In particular, I want to improve further in:

- React architecture
- schema-driven UI systems
- frontend maintainability
- accessibility-aware engineering
- test-heavy frontend development
- review-driven iteration

OpenEverest’s V2 plugin ecosystem is especially interesting because it combines runtime systems thinking with frontend implementation work.

---

### 2. Real-World CSP & Security Engineering Experience

One of the most interesting parts of this project is that the editor integration must work inside a strict production CSP environment.

That introduces real engineering constraints around:

- runtime evaluation
- style injection
- editor architecture
- nonce propagation
- production security guarantees

I specifically want hands-on experience designing frontend tooling that operates safely under strict CSP requirements instead of relying on relaxed development assumptions.

---

### 3. Long-Term Open Source Growth

I also want to continue contributing beyond the official mentorship period.

My long-term goal is to remain involved in:

- `area/ui`
- plugin tooling
- schema-driven workflows
- frontend testing
- developer experience improvements

I want the work done during the mentorship to become part of a maintainable long-term subsystem rather than a short-lived experimental feature.

---

### Additional Long-Term Goal

Another important goal is helping establish reusable frontend patterns for the V2 plugin ecosystem itself.

Particularly around:

- secure editor integrations
- validation workflows
- reusable playground infrastructure
- CSP-safe frontend tooling
- testing conventions
- contributor onboarding

The hope is that future contributors working in the same subsystem inherit a maintainable foundation instead of needing to restart architectural exploration from scratch every time.

---

# 1. Title & Metadata

| Field | Value |
|---|---|
| Project | OpenEverest |
| Organization | CNCF (OpenEverest project) |
| Mentorship Title | Plugin Developer Playground: Interactive UI Schema Editor with Live Preview (V2) |
| Upstream Tracking Issue | `openeverest/openeverest#2059` |
| Term | 2026 Term 2 (Jun–Aug 2026) |
| Applicant | Dev Jaiswal |
| GitHub | `@cynox-66` |
| Timezone | IST (UTC+5:30) |
| Weekly Availability | 30 hours/week |
| Mentors | Iaroslavna Soloveva (`@solovevayaroslavna`), Sergey Pronin (`@spron-in`) |
| Proposal Version | v1 |
| Date | 2026-05-13 |

---

# 2. Abstract

OpenEverest V2 allows database providers to define their UIs declaratively through YAML schemas rendered dynamically by the `UIGenerator` runtime.

However, plugin authors currently lack an efficient development workflow for iterating on these schemas.

Today, testing schema changes requires repeatedly:

- updating a Provider CRD
- applying it to Kubernetes
- refreshing the UI
- navigating back to the correct workflow

This makes iteration slow, infrastructure-heavy, and difficult to debug.

---

## Existing Problem

An earlier proof-of-concept at:

`pages/ui-generator-builder/`

attempted to provide a live editor + preview workflow using Monaco Editor.

However, Monaco depends on runtime evaluation patterns that violate OpenEverest’s strict production CSP.

Specifically:

- `new Function()`
- worker-based execution
- runtime dynamic evaluation

are blocked by the current security policy.

As a result, the existing POC is not production deployable.

---

## Proposed Solution

This proposal introduces a new:

`/plugin-developer`

playground built around:

- CodeMirror 6
- live UIGenerator preview
- inline YAML validation
- Zod-backed schema diagnostics
- mock provider support
- local persistence
- `.yaml` import/export workflows

while preserving the existing production CSP unchanged.

---

## Core Architectural Direction

The playground intentionally reuses the existing V2 runtime directly instead of building a parallel rendering system.

The right-hand preview pane uses:

- `UIGenerator`
- `useFormEngine`
- `preprocessComponent`
- `postprocessSchemaData`
- `ApiProviderRegistry`

This ensures that playground behavior remains aligned with production runtime behavior.

---

## Existing Validation Work

As part of proposal preparation, I also implemented a CSP-safe local proof-of-concept directly against the V2 branch.

The PoC successfully validated:

- CodeMirror 6 integration under strict CSP
- nonce propagation through the existing frontend architecture
- inline YAML diagnostics
- split-pane rendering
- route-level integration
- production-build compatibility
- zero CSP violations
- initial unit/component test coverage

This early validation significantly reduced implementation uncertainty before the mentorship period begins.

---

## Final Deliverables

The project ultimately delivers:

- a production-safe schema playground
- live runtime preview
- inline diagnostics
- mock provider infrastructure
- persistence workflows
- accessibility-aware editor interactions
- CSP-safe E2E validation
- removal of the legacy Monaco-based POC

while preserving OpenEverest’s existing runtime and security architecture unchanged.

# 3. Background & Why This Work Matters

---

## The OpenEverest V2 Plugin Model

OpenEverest V2 introduces a plugin architecture where database providers define their UI workflows declaratively through YAML schemas embedded inside a Provider CRD.

These schemas are rendered dynamically at runtime through the existing frontend infrastructure.

Core runtime components already include:

- `UIGenerator`
- `useFormEngine`
- Zod + CEL validation
- preprocessing/postprocessing pipelines
- provider-backed dynamic fields
- topology-aware form orchestration

The architecture is already heavily schema-driven.

Because of this, the missing piece is not rendering capability itself.

The missing piece is a secure, developer-friendly workflow for authoring and previewing schemas efficiently.

---

## Current Developer Workflow Problems

Today, plugin authors must repeatedly:

1. modify a schema
2. update a Provider CRD
3. apply changes to Kubernetes
4. reload the UI
5. navigate back to the correct form
6. reproduce the workflow again

This creates a slow feedback loop for even small schema changes.

The result is:

- slower iteration
- harder debugging
- increased contributor friction
- reduced experimentation speed

As the V2 plugin ecosystem grows, this workflow becomes increasingly difficult to scale.

---

## Existing Playground Proof-of-Concept

An existing proof-of-concept already exists at:

`pages/ui-generator-builder/`

The POC demonstrates that a:

- live YAML editor
- real-time preview
- schema-driven workflow

is technically feasible.

However, the implementation currently depends on Monaco Editor.

That creates a major production blocker.

---

## The CSP Problem

OpenEverest enforces a strict production Content Security Policy (CSP).

The existing CSP intentionally blocks unsafe runtime execution patterns such as:

- `unsafe-eval`
- unrestricted worker execution
- arbitrary dynamic script evaluation

Monaco internally depends on runtime behaviors like:

- `new Function()`
- worker-based execution
- dynamic evaluation infrastructure

As a result, the current playground does not function correctly under the real production CSP.

This means the existing POC cannot safely ship in production.

---

## Why This Problem Matters

This is not simply an editor replacement issue.

The real challenge is:

> creating a fast developer workflow without weakening the platform’s security guarantees.

The project therefore sits at the intersection of:

- frontend systems engineering
- runtime architecture
- browser security
- developer tooling
- OSS maintainability

---

## Existing Runtime Infrastructure Already Solves Most Problems

One important architectural observation is that OpenEverest V2 already contains most of the runtime infrastructure required for a production playground.

The runtime already supports:

### Schema-Driven Rendering

Through:

- `UIGenerator`
- `useFormEngine`

---

### Validation Infrastructure

Including:

- Zod validation
- CEL validation
- runtime preprocessing

---

### Dynamic Provider-Backed Fields

Through:

- `ApiProviderRegistry`

---

### Runtime Transformation Pipelines

Including:

- `preprocessComponent`
- `postprocessSchemaData`

---

### Topology-Aware Form Flows

Supporting:

- conditional rendering
- multi-step workflows
- runtime state orchestration

---

## Key Architectural Insight

Because these runtime systems already exist, the playground does **not** need to implement a parallel rendering engine.

Instead, it can reuse the existing runtime directly.

That becomes one of the most important architectural decisions in this proposal.

---

# Why Existing Runtime Reuse Matters

Reusing the production runtime provides several advantages immediately.

### Benefits

- runtime parity
- lower maintenance overhead
- shared validation behavior
- earlier bug discovery
- reduced architectural duplication

---

### Avoids

- duplicated renderer logic
- playground/runtime drift
- inconsistent preprocessing behavior
- separate validation systems

---

## Who Benefits From This Work?

The playground benefits multiple groups simultaneously.

---

## 1. Plugin Authors

Plugin authors gain a dramatically faster development workflow.

Instead of relying on Kubernetes deployment cycles for every schema edit, they can iterate directly inside a live preview environment.

This enables:

- rapid experimentation
- easier debugging
- faster schema authoring
- lower onboarding friction

---

### Workflow Improvement

The workflow shifts from:

`cluster-driven iteration`

to:

`direct schema-driven iteration`

That is a major productivity improvement.

---

## 2. Maintainers

Maintainers gain a reproducible runtime debugging environment tied directly to the production rendering system.

Instead of debugging screenshots or incomplete reproductions, maintainers can validate schemas immediately against:

- the real runtime
- the real validation flow
- the real preprocessing pipeline
- the real UIGenerator implementation

This improves:

- reproducibility
- review quality
- contributor onboarding
- debugging efficiency

---

## 3. Documentation Contributors

The playground also enables executable documentation workflows.

Instead of static YAML snippets, future documentation can potentially include:

- interactive examples
- reproducible playground flows
- live schema demonstrations
- editable tutorials

Over time, the playground can become part of the broader developer experience ecosystem rather than remaining only a debugging utility.

---

# Industry Context: Monaco vs Strict CSP

The Monaco/CSP conflict is not unique to OpenEverest.

A widely referenced example is:

`grafana/grafana#51047`

which documents similar struggles integrating Monaco into strict CSP environments.

The underlying issue is architectural.

### Monaco Depends On

- runtime code generation
- dynamic evaluation
- worker-based execution

### Strict CSP Intentionally Blocks

- `eval()`
- `new Function()`
- unrestricted workers

This creates a fundamental compatibility problem.

---

## Why CodeMirror 6 Emerged as the Preferred Direction

Multiple contributors discussing Issue `#2059` independently converged on CodeMirror 6 as the most practical replacement approach.

The reasons are architectural.

CodeMirror 6:

- avoids `unsafe-eval`
- avoids Monaco-style worker infrastructure
- supports nonce-safe style injection
- works cleanly under strict CSP environments

This proposal builds on that architectural direction and validates it directly against the real OpenEverest frontend environment.

---

# Existing Proof-of-Concept Validation

A major part of proposal preparation involved validating the highest-risk assumptions before implementation begins.

A local CSP-safe proof-of-concept was implemented directly against the V2 branch.

The goal was not to build the final playground early.

The goal was to validate the architecture itself.

---

## The PoC Successfully Validated

| Assumption | Result |
|---|---|
| CodeMirror 6 works under the existing CSP | Confirmed |
| Existing nonce propagation flow is reusable | Confirmed |
| Backend CSP changes are unnecessary | Confirmed |
| Existing UIGenerator runtime can power live preview | Confirmed |
| Route-level integration is straightforward | Confirmed |
| Inline YAML diagnostics are feasible | Confirmed |
| Existing testing patterns adapt cleanly | Confirmed |

---

## Additional PoC Results

The prototype also demonstrated:

- split-pane rendering
- inline diagnostics
- route integration
- runtime compatibility
- production preview functionality
- accessibility-aware keyboard handling
- zero CSP violations
- initial unit/component testing

This significantly reduced implementation uncertainty before the mentorship period begins.

---

# Why This Work Is Valuable Long-Term

The playground is more than a convenience feature.

It has the potential to become:

- the primary onboarding surface for plugin authors
- a validation environment for schema workflows
- a debugging surface for maintainers
- a foundation for future tooling
- part of the documentation ecosystem

That is why this proposal prioritizes:

- runtime parity
- maintainability
- security correctness
- testability
- incremental extensibility

instead of focusing only on visual functionality.

# 4. Current Architecture Analysis

---

## How the V2 Plugin Runtime Works Today

OpenEverest V2 already contains most of the infrastructure required for a schema playground.

The missing layer is not rendering capability.

The missing layer is a secure, developer-facing editing workflow integrated into the existing runtime.

---

## High-Level Runtime Flow

A Provider CRD contains a:

`uiSchema`

field of type:

`TopologyUISchemas`

When a user opens a database creation flow, the runtime executes a multi-stage pipeline.

---

## Runtime Pipeline

1. fetch Provider CRD
2. parse provider schema
3. preprocess schema components
4. initialize form engine
5. render through `UIGenerator`
6. postprocess submitted values

---

## Core Runtime Components

The existing V2 architecture already provides several reusable runtime primitives.

---

## `UIGenerator`

The runtime renderer responsible for converting declarative schemas into interactive React form flows.

Responsibilities include:

- topology rendering
- component orchestration
- dynamic field rendering
- section composition
- runtime form integration

For the playground, this becomes the right-hand live preview pane.

---

## `useFormEngine`

The form orchestration layer already handles:

- validation lifecycle
- dependency resolution
- conditional rendering
- form state management
- topology navigation
- submission orchestration

This is important because the playground automatically inherits production runtime behavior instead of duplicating it.

---

## `ApiProviderRegistry`

Provider-backed fields already resolve through the shared registry infrastructure.

Examples include:

- storage classes
- monitoring providers
- backup providers
- dynamic runtime options

The registry already exposes a reusable provider lifecycle abstraction.

This becomes especially important for mock-provider integration later in the proposal.

---

## Runtime Processing Pipeline

The existing architecture already includes:

- `preprocessComponent`
- `postprocessSchemaData`

These runtime transforms contain important production assumptions around:

- field normalization
- dynamic path handling
- runtime transformations
- payload shaping

Reusing them directly inside the playground ensures:

- runtime parity
- validation consistency
- lower maintenance burden

---

# Existing Runtime Reuse Strategy

A major architectural decision in this proposal is:

> reuse the production runtime directly instead of building parallel playground abstractions.

---

## Why This Matters

Duplicating rendering infrastructure would introduce:

- runtime drift
- inconsistent validation behavior
- duplicate maintenance burden
- debugging inconsistencies

Using the real runtime instead ensures that plugin authors preview against the same execution flow production actually uses.

---

# Existing CSP Architecture

The current OpenEverest CSP implementation is intentionally strict.

The policy blocks:

- `unsafe-eval`
- unrestricted inline scripts
- arbitrary worker execution
- unsafe iframe execution paths

This is one of the most important architectural constraints in the project.

---

## Current Nonce Propagation Flow

The frontend already propagates CSP nonces through the application runtime.

The flow currently looks like:

```txt
Go Middleware
    ↓
CSP Nonce
    ↓
<meta name="csp-nonce">
    ↓
React Runtime
    ↓
Emotion Style Injection
```

This existing architecture becomes the foundation for the CodeMirror integration.

---

## Important Architectural Discovery

One of the most important findings during proposal preparation was that:

> CodeMirror 6 can integrate cleanly into the existing nonce propagation flow without requiring CSP policy changes.

That significantly reduces implementation risk.

---

# Problems With the Existing Playground POC

Although the current proof-of-concept demonstrates the correct high-level workflow, several major architectural problems still exist.

---

# 1. Monaco Violates the Production CSP

The existing POC imports:

`@monaco-editor/react`

Monaco internally depends on:

- runtime code generation
- `new Function()`
- worker-based execution
- dynamic evaluation infrastructure

These behaviors are intentionally blocked by the current CSP.

---

## Why This Matters

The current playground therefore:

- fails under production CSP
- cannot safely ship
- would require security relaxation to deploy

The proposal intentionally avoids weakening the existing security model.

---

# Why CodeMirror 6 Is Better Aligned

CodeMirror 6:

- avoids `unsafe-eval`
- avoids runtime dynamic evaluation
- avoids Monaco-style workers
- supports nonce-safe style injection
- works under strict CSP environments

This becomes the central architectural direction behind the proposal.

---

# 2. No Structural Schema Validation

The current POC validates YAML syntax only.

It does not validate schema structure against:

`TopologyUISchemas`

This creates several problems.

---

## Current Problems

- malformed schemas partially render
- invalid structures silently fail
- diagnostics are weak
- maintainers must debug runtime behavior manually

---

## Proposed Improvement

The new playground introduces:

- Zod-backed structural validation
- inline diagnostics
- editor lint markers
- runtime-aware validation
- preprocess smoke validation

This transforms the playground into a true schema-authoring environment instead of a basic text editor.

---

# 3. No Persistence Layer

Refreshing the current playground destroys all work.

This prevents:

- iterative schema development
- reusable playground workflows
- debugging continuity
- local experimentation

---

## Proposed Persistence Features

The proposal adds:

- autosave
- named schemas
- local persistence
- schema restoration
- `.yaml` import/export

This turns the playground into a practical developer workflow instead of a temporary demo page.

---

# 4. No Automated Testing Coverage

The existing POC has effectively no meaningful test coverage.

This contrasts heavily with the surrounding V2 architecture, which already emphasizes:

- colocated tests
- runtime validation tests
- frontend interaction testing
- integration coverage

---

## Why This Matters

The playground interacts directly with:

- runtime rendering
- validation systems
- provider registries
- CSP-sensitive editor infrastructure

Without testing, regressions become difficult to detect safely.

---

## Proposed Testing Direction

The proposal introduces:

- unit tests
- component tests
- integration tests
- CSP-specific E2E tests
- accessibility assertions
- registry lifecycle testing

The accompanying proof-of-concept already validated this direction with initial passing test coverage.

---

# 5. No Mock Provider Support

Provider-backed fields currently require real backend infrastructure to preview correctly.

This creates unnecessary friction during schema development.

---

## Current Problem

Plugin authors cannot easily prototype fields backed by:

- provider registries
- runtime option loaders
- dynamic select sources

without implementing real APIs.

---

## Proposed Solution

The playground introduces inline:

`mockData:`

support.

This allows temporary provider implementations to register dynamically into the existing provider registry during preview rendering.

---

## Why This Approach Matters

The implementation intentionally preserves the real runtime flow.

The playground still uses:

- the real registry
- the real hooks
- the real provider resolution lifecycle

This provides much higher runtime fidelity than a mocked parallel system.

---

# 6. No Dedicated Developer Documentation

Current UIGenerator documentation focuses primarily on runtime architecture.

It does not currently document:

- schema playground workflows
- CSP editor constraints
- validation debugging
- mock-provider usage
- persistence flows

---

## Proposed Documentation Additions

The proposal adds dedicated playground documentation covering:

- setup workflows
- validation behavior
- CSP architecture
- import/export workflows
- mock providers
- troubleshooting
- extension guidance

This is important for long-term contributor sustainability.

---

# Key Architectural Conclusion

After analyzing the existing runtime architecture, one conclusion became very clear:

> OpenEverest already contains most of the runtime systems required for a production playground.

The primary missing pieces are:

- CSP-safe editor infrastructure
- validation tooling
- persistence workflows
- mock-provider support
- developer-focused UX

Because of this, the implementation can remain:

- additive
- runtime-aligned
- security-safe
- incrementally reviewable

without requiring major architectural rewrites.

# 5. Proposed Solution

---

## Plain-Language Summary

The proposed playground is intentionally designed as a thin developer layer on top of the existing V2 runtime.

It does **not** introduce a parallel rendering system.

Instead, the right-hand preview pane directly reuses:

- `UIGenerator`
- `useFormEngine`
- `preprocessComponent`
- `postprocessSchemaData`
- `ApiProviderRegistry`

This ensures plugin authors preview schemas against the same runtime behavior used in production.

---

## High-Level Playground Workflow

The playground introduces four major capabilities:

1. a CSP-safe YAML editor
2. runtime-aware schema validation
3. mock provider support
4. persistence + import/export workflows

All while preserving the existing production CSP unchanged.

---

# Core Architecture

The runtime architecture intentionally reuses existing production systems wherever possible.

---

## High-Level Runtime Flow

```txt
CodeMirror Editor
        ↓
YAML Parser
        ↓
Zod Validation
        ↓
Mock Provider Registrar
        ↓
useFormEngine
        ↓
UIGenerator
        ↓
postprocessSchemaData
        ↓
Output Preview
```

---

## Key Architectural Principle

The playground does not attempt to emulate the production runtime.

It uses the real runtime directly.

That decision minimizes:

- runtime drift
- duplicate logic
- validation inconsistencies
- long-term maintenance burden

---

# CSP Architecture

The existing OpenEverest CSP policy remains unchanged.

No new security exceptions are introduced.

---

## Existing Nonce Flow Reuse

The editor reuses the same nonce propagation architecture already used by Emotion.

Current flow:

```txt
Go Middleware
    ↓
CSP Nonce
    ↓
<meta name="csp-nonce">
    ↓
React Runtime
    ↓
CodeMirror Style Injection
```

---

## Why This Matters

The proposal intentionally avoids:

- `unsafe-eval`
- iframe workarounds
- CSP relaxation
- blob worker permissions
- runtime code generation

The editor architecture remains fully aligned with the existing production security model.

---

# Main Playground Components

The playground introduces four focused infrastructure layers.

---

# 1. CodeMirror 6 Editor

The Monaco-based POC is replaced with a CSP-safe CodeMirror 6 integration.

---

## Responsibilities

The editor layer provides:

- YAML editing
- syntax highlighting
- inline diagnostics
- keyboard accessibility
- lint rendering
- runtime-safe style injection

---

## Why CodeMirror 6

CodeMirror 6 was selected because it:

- avoids `unsafe-eval`
- avoids Monaco-style workers
- supports nonce-safe style injection
- works under strict CSP
- remains lightweight and modular

---

## Important Tradeoff

CodeMirror ships with fewer built-in features than Monaco.

This proposal intentionally accepts that tradeoff in exchange for:

- CSP compatibility
- simpler runtime architecture
- lower security risk
- easier long-term maintenance

---

# 2. YAML + Zod Validation Layer

The playground validates schemas structurally instead of only validating YAML syntax.

---

## Validation Pipeline

```txt
YAML Parse
    ↓
Structural Validation
    ↓
Runtime Smoke Validation
    ↓
Inline Diagnostics
```

---

## Validation Goals

The validation system provides:

- line-aware diagnostics
- runtime-aware validation
- editor lint markers
- preprocessing smoke checks
- resilient partial-edit handling

---

## Validation Philosophy

The proposal intentionally validates:

### Strictly

- topology structure
- sections
- ordering arrays
- required object shapes

### Loosely

- nested provider-specific extensions
- component internals
- future plugin-specific fields

This preserves compatibility with the existing runtime flexibility.

---

# 3. Mock Provider Infrastructure

The playground supports provider-backed field prototyping without requiring backend APIs.

---

## Example

```
dataSource:
  provider: storage-classes
  mockData:
    - label: fast-storage
      value: fast-storage
```

---

## Why This Matters

Plugin authors can prototype:

- select fields
- provider-backed options
- dynamic runtime behaviors

without implementing real infrastructure first.

---

## Registry Lifecycle

The implementation dynamically:

1. extracts mock providers
2. registers them
3. refreshes them during edits
4. unregisters stale entries

This preserves runtime accuracy during live preview updates.

---

## Important Architectural Decision

The playground reuses the real:

`ApiProviderRegistry`

instead of introducing a playground-only abstraction.

Benefits include:

- runtime parity
- lower maintenance burden
- shared execution paths
- earlier bug discovery

---

# 4. Persistence + Import/Export

The playground supports lightweight client-side persistence.

---

## Features

- autosave
- named schemas
- local persistence
- restore flows
- `.yaml` import/export

---

## Why Persistence Matters

Without persistence:

- experimentation becomes fragile
- workflows reset on refresh
- debugging becomes difficult
- onboarding friction increases

Persistence transforms the playground from a temporary editor into a usable developer workflow.

---

# Runtime Preview

The right-hand preview pane renders schemas using the real production runtime.

---

## Runtime Components Reused

The preview directly integrates:

- `UIGenerator`
- `useFormEngine`
- `FormMode`
- preprocessing pipelines
- postprocessing pipelines

---

## Why Runtime Reuse Is Critical

Any duplicated rendering abstraction would introduce:

- runtime drift
- inconsistent validation behavior
- duplicate maintenance
- debugging confusion

Using the real runtime directly guarantees production parity.

---

# Output Panel

The playground also includes a processed output viewer.

---

## Responsibilities

The output panel displays:

- transformed form values
- processed payload structure
- postprocessed runtime output
- final submission-ready data

This allows plugin authors to validate not only rendering behavior but also submission semantics.

---

# Backward Compatibility

The implementation is intentionally additive.

---

## Added

- `/plugin-developer`
- CodeMirror integration
- validation infrastructure
- persistence workflows
- mock-provider support

---

## Removed

- legacy Monaco-based POC

---

## Unchanged

- production runtime behavior
- CSP policy
- backend contracts
- provider APIs
- authentication flow

No existing production workflows are broken.

---

# Security Model

The proposal keeps the existing CSP policy fully intact.

No additional directives are introduced.

---

## Specifically Avoided

The implementation avoids:

- `unsafe-eval`
- inline script exceptions
- iframe sandbox hacks
- dynamic runtime evaluation
- Monaco worker infrastructure

---

## Why This Matters

The proposal treats CSP compatibility as a first-class architectural requirement rather than an afterthought.

That is one of the defining goals of the entire project.

---

# Existing Proof-of-Concept Validation

A local CSP-safe proof-of-concept was already implemented before proposal submission.

This validation work confirmed:

- CodeMirror compatibility
- nonce propagation
- production preview functionality
- route integration
- validation feasibility
- split-pane rendering
- runtime reuse viability

---

## Additional PoC Results

The prototype also demonstrated:

- zero CSP violations
- accessibility-aware keyboard handling
- inline diagnostics
- route-level integration
- initial unit/component testing

This significantly reduced architectural uncertainty before implementation begins.

---

# Error Handling Philosophy

The playground separates failures into two categories.

---

## User-Facing Validation Failures

Handled inline inside the editor.

Examples:

- YAML syntax errors
- invalid topology structure
- malformed mock providers
- validation failures

These surface as diagnostics instead of crashing the application.

---

## Unexpected Runtime Failures

Examples include:

- runtime preprocessing exceptions
- provider registry corruption
- form engine failures

These bubble into the existing application-level error boundaries already present in the V2 runtime.

The proposal intentionally avoids creating a parallel error system.

---

# Key Architectural Advantages

The proposed architecture provides several major benefits simultaneously.

---

## Security

- strict CSP preserved
- no security relaxation
- no unsafe runtime evaluation

---

## Runtime Accuracy

- real UIGenerator reuse
- shared validation behavior
- shared processing pipelines

---

## Maintainability

- additive architecture
- minimal duplication
- reuse-first design philosophy

---

## Contributor Experience

- faster iteration
- local experimentation
- reproducible debugging
- persistence workflows

---

## Long-Term Extensibility

The architecture intentionally leaves room for future improvements such as:

- schema autocomplete
- richer diagnostics
- reusable editor packages
- CLI validation parity
- embedded documentation examples

without requiring architectural rewrites later.

# 6. Technical Implementation Plan

---

## Implementation Philosophy

All implementation work targets the existing V2 architecture directly.

The proposal intentionally follows the repository’s current frontend patterns:

- additive architecture
- colocated tests
- small reviewable PRs
- runtime reuse
- CSP-safe rendering
- incremental feature slices
- minimal abstraction layering

The implementation is designed so every phase can be reviewed independently without maintainers needing to reason about the entire playground stack simultaneously.

---

# Repository Scope

Unless otherwise specified, all implementation paths refer to:

```txt
ui/apps/everest/src/
```

on the `v2` branch.

---

# Implementation Strategy

The implementation sequence prioritizes:

1. security-sensitive architecture first
2. runtime integration second
3. developer workflow tooling third
4. persistence and polish last

This reduces the risk of late-stage architectural rewrites.

---

# Planned Implementation Slices

| Phase | Primary Goal |
|---|---|
| 6.1 | Route shell + split-pane infrastructure |
| 6.2 | CodeMirror 6 integration |
| 6.3 | YAML parsing + Zod validation |
| 6.4 | Mock-provider infrastructure |
| 6.5 | Runtime preview integration |
| 6.6 | Persistence layer |
| 6.7 | Import/export workflows |
| 6.8 | Legacy POC removal |

---

# 6.1 Route Shell & Split-Pane Infrastructure

---

## Goals

The first implementation slice establishes the foundational playground structure.

This phase intentionally avoids editor complexity so layout architecture can be reviewed independently.

---

## Responsibilities

The initial infrastructure includes:

- route registration
- page-level orchestration
- split-pane layout
- responsive resizing
- toolbar shell
- isolated rendering boundaries

---

# New Files

---

## Main Page Container

```txt
pages/plugin-developer/plugin-developer.tsx
```

Primary orchestration layer responsible for:

- editor state
- selected schema state
- validation coordination
- preview composition
- toolbar actions
- persistence coordination

The component intentionally avoids embedding editor internals directly.

---

## Shared Playground Types

```txt
pages/plugin-developer/plugin-developer.types.ts
```

Contains shared domain types:

```ts
type PlaygroundState
type PersistedSchema
type LintDiagnostic
type MockProviderEntry
```

Centralized types reduce cross-module coupling.

---

## Barrel Export

```txt
pages/plugin-developer/index.ts
```

Provides stable imports for route-level lazy loading.

---

## Split-Pane Component

```txt
pages/plugin-developer/split-pane/split-pane.tsx
```

Reusable split-pane infrastructure using CSS Grid.

---

## Split-Pane Responsibilities

- draggable divider
- responsive resizing
- pointer handling
- touch support
- resize constraints
- listener cleanup
- future persistence compatibility

---

# Initial Test Files

```txt
pages/plugin-developer/plugin-developer.test.tsx
pages/plugin-developer/split-pane/split-pane.test.tsx
```

---

## Initial Test Coverage

The first slice validates:

- route rendering
- split-pane behavior
- resize cleanup
- responsive layout handling
- keyboard accessibility
- pointer lifecycle cleanup

---

# Route Registration

---

## Additive Route Entry

```tsx
{
  path: 'plugin-developer',
  element: withSuspense(<PluginDeveloper />),
}
```

The route remains additive throughout development to minimize regression risk.

---

# Legacy POC Strategy

The existing:

```txt
/ui-generator-builder
```

route is retained temporarily during early implementation slices.

Removal occurs only after functional parity is reached.

This prevents destabilizing existing development workflows prematurely.

---

# Design Decisions

---

## Why a Dedicated Route?

A standalone route provides:

- isolated testing
- simpler screenshots/GIFs
- cleaner Playwright flows
- easier contributor onboarding
- independent iteration
- future documentation embedding support

---

## Why Not Extend UIGenerator Directly?

The playground is a developer workflow surface, not a runtime rendering primitive.

Keeping it route-scoped avoids tightly coupling:

- persistence logic
- diagnostics infrastructure
- authoring UX
- mock-provider behavior

to the production runtime itself.

---

# 6.2 CodeMirror 6 Integration

---

## Goals

Provide a reusable editor abstraction that:

- works under strict CSP
- supports inline diagnostics
- integrates with MUI theming
- remains lightweight
- supports future extensibility
- preserves accessibility requirements

---

# New Files

---

## Editor Wrapper

```txt
components/code-editor/code-editor.tsx
```

Primary CodeMirror integration component.

---

## Responsibilities

The editor wrapper handles:

- editor lifecycle management
- nonce propagation
- extension registration
- diagnostics rendering
- synchronization behavior
- cleanup on unmount

---

## Extension Builder

```txt
components/code-editor/extensions.ts
```

Responsible for reusable extension composition.

This keeps editor configuration declarative and easier to evolve incrementally.

---

## Theme Integration

```txt
components/code-editor/theme.ts
```

Defines editor styling aligned with the existing MUI theme system.

Includes:

- dark mode
- light mode
- focus styling
- lint highlighting
- selection behavior

---

# Editor Test Files

```txt
components/code-editor/code-editor.test.tsx
components/code-editor/extensions.test.ts
```

---

## Editor Test Coverage

Coverage includes:

- initialization behavior
- diagnostics rendering
- update synchronization
- nonce propagation
- keyboard behavior
- cleanup semantics
- extension registration

---

# Editor Component API

```ts
type CodeEditorProps = {
  value: string;
  onChange: (value: string) => void;
  language: 'yaml';
  diagnostics: Diagnostic[];
  readOnly?: boolean;
  nonce: string;
};
```

The API intentionally remains minimal.

This avoids premature over-generalization.

---

# Planned Extension Stack

```ts
import { yaml } from '@codemirror/lang-yaml';

import {
  autocompletion,
  completionKeymap,
  closeBrackets,
} from '@codemirror/autocomplete';

import {
  defaultKeymap,
  history,
  historyKeymap,
  indentWithTab,
} from '@codemirror/commands';

import {
  bracketMatching,
  foldGutter,
  indentOnInput,
} from '@codemirror/language';

import {
  linter,
  lintGutter,
  lintKeymap,
} from '@codemirror/lint';

import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
} from '@codemirror/view';

import {
  searchKeymap,
  highlightSelectionMatches,
} from '@codemirror/search';
```

---

# Extension Philosophy

The extension stack intentionally remains conservative.

The proposal avoids:

- heavyweight language servers
- worker-backed completion systems
- Monaco-style runtime services
- unnecessary plugins

This keeps the editor:

- lightweight
- maintainable
- CSP-safe

---

# Extension Composition

```ts
export const buildExtensions = (opts: {
  diagnosticsSource: (
    view: EditorView
  ) => readonly Diagnostic[];

  onChange: (value: string) => void;
}) => [
  lineNumbers(),
  foldGutter(),
  highlightActiveLine(),
  highlightSelectionMatches(),

  history(),
  bracketMatching(),
  closeBrackets(),
  indentOnInput(),

  yaml(),

  autocompletion(),

  linter(opts.diagnosticsSource, {
    delay: 250,
  }),

  lintGutter(),

  keymap.of([
    ...defaultKeymap,
    ...historyKeymap,
    ...searchKeymap,
    ...completionKeymap,
    ...lintKeymap,
    indentWithTab,
  ]),

  EditorView.updateListener.of((u) => {
    if (u.docChanged) {
      opts.onChange(
        u.state.doc.toString()
      );
    }
  }),
];
```

---

# CSP Nonce Integration

This is the most security-sensitive part of the implementation.

The playground reuses the existing nonce already injected into the app shell.

---

## Existing Nonce Source

```html
<meta name="csp-nonce" content="{{.CSPNonce}}" />
```

---

## Runtime Nonce Extraction

```ts
const nonce =
  document
    .querySelector(
      "meta[name='csp-nonce']"
    )
    ?.getAttribute('content') || '';
```

---

## CodeMirror Nonce Wiring

```ts
EditorView.cspNonce.of(nonce)
```

This ensures dynamically injected styles remain compliant with the production CSP.

---

# Why CodeMirror Works Under Strict CSP

CodeMirror 6:

- avoids `eval()`
- avoids `new Function()`
- avoids blob workers
- uses static parser generation
- supports nonce-safe style injection

This aligns naturally with OpenEverest’s security model.

---

# Why Monaco Was Rejected

Monaco relies heavily on:

- dynamic runtime evaluation
- worker infrastructure
- blob execution paths
- runtime code generation

Supporting Monaco would require weakening the current CSP guarantees.

The proposal intentionally prioritizes security alignment over feature richness.

---

# Accessibility Considerations

Accessibility is treated as a production requirement.

The editor supports:

- keyboard navigation
- focus visibility
- tab indentation
- lint navigation
- screen-reader compatibility where supported
- predictable focus recovery

Accessibility validation is integrated directly into the test strategy.

# 6.3 YAML Parsing & Zod Validation

---

## Goals

The validation layer provides:

- immediate editor feedback
- structural schema validation
- runtime-aware diagnostics
- inline lint annotations
- resilient partial-edit handling

without blocking typing responsiveness.

The implementation intentionally validates against the real runtime shape instead of introducing playground-only schema semantics.

---

# New Files

---

## YAML Parser Wrapper

```txt
pages/plugin-developer/validator/yaml-parser.ts
```

Responsible for:

- YAML parsing
- line-aware error handling
- parser diagnostics
- unsupported document detection

---

## Zod Validation Layer

```txt
pages/plugin-developer/validator/topology-ui-schemas.zod.ts
```

Defines runtime-compatible validation for:

```ts
TopologyUISchemas
```

---

## Diagnostic Conversion Layer

```txt
pages/plugin-developer/validator/to-diagnostics.ts
```

Converts parser and validation failures into:

```ts
Diagnostic[]
```

objects compatible with CodeMirror lint rendering.

---

# Validation Test Files

```txt
yaml-parser.test.ts
topology-ui-schemas.zod.test.ts
to-diagnostics.test.ts
```

---

## Validation Test Coverage

Coverage includes:

- valid schemas
- malformed YAML
- invalid topology structures
- partial schema recovery
- multiline parsing behavior
- diagnostic range mapping
- nested validation paths

---

# YAML Parsing Layer

---

## Parsing Strategy

The parser wraps:

```ts
yaml.parse()
```

together with:

```ts
LineCounter
```

to preserve accurate line and column information.

This allows diagnostics to map cleanly into editor ranges.

---

# Why Line-Aware Parsing Matters

Without line-aware parsing:

- diagnostics become vague
- inline lint markers become unreliable
- debugging malformed schemas becomes frustrating

Accurate positioning is essential for a usable editing experience.

---

# Parse Flow

```text
Editor Text
    │
    ▼
yaml.parse()
    │
    ├── Parse failure
    │       ▼
    │   Diagnostic[]
    │
    ▼
Parsed Object
```

---

# Multi-Document Rejection

The playground intentionally rejects YAML streams containing multiple documents.

Example:

```yaml
---
doc1
---
doc2
```

---

## Why Multi-Document YAML Is Rejected

Supporting multiple documents introduces ambiguity around:

- topology ownership
- validation semantics
- preview routing
- persistence behavior

The playground intentionally optimizes for a single-schema workflow.

---

# Zod Validation Layer

---

## Validation Philosophy

The proposal validates:

- structure strictly
- nested component internals loosely

This mirrors the flexibility already present in the runtime type system.

The implementation intentionally avoids over-constraining future provider schemas.

---

# Example Section Schema

```ts
export const sectionSchema = z.object({
  label: z.string().optional(),

  description: z.string().optional(),

  components: z.record(
    z.string(),
    z.unknown()
  ),

  componentsOrder:
    z.array(z.string()).optional(),
});
```

---

# Topology Schema

```ts
export const topologySchema =
  z.object({
    sections: z.record(
      z.string(),
      sectionSchema
    ),

    sectionsOrder:
      z.array(z.string()).optional(),
  }).passthrough();
```

---

# Top-Level Schema

```ts
export const topologyUISchemas =
  z.record(
    z.string(),
    topologySchema
  ).and(
    z.record(
      z.string(),
      z.unknown()
    )
  );
```

---

# Why Validation Remains Flexible

`TopologyUISchemas` intentionally includes:

```ts
Record<string, unknown>
```

escape hatches.

Overly strict validation would reject:

- future provider extensions
- downstream custom fields
- experimental schema patterns

even if the runtime itself would still accept them.

---

# Runtime Smoke Validation

After successful Zod validation:

```ts
preprocessComponent()
```

runs inside:

```ts
try/catch
```

to surface unexpected runtime assumptions safely.

---

# Why a Smoke Pass Exists

The runtime already contains preprocessing assumptions around:

- field normalization
- dynamic paths
- component transformations
- provider resolution

The smoke pass helps catch incompatibilities early without crashing the UI.

---

# Failure Handling Philosophy

Unexpected runtime failures become:

```txt
warning diagnostics
```

instead of fatal application crashes.

This preserves editor usability during incomplete edits.

---

# Diagnostics Conversion

Validation failures convert into CodeMirror-compatible diagnostics.

---

## Diagnostic Shape

```ts
{
  from,
  to,
  severity,
  message
}
```

---

# Diagnostic Features

This enables:

- gutter markers
- inline underlines
- hover messages
- keyboard navigation
- severity-aware rendering

directly inside the editor.

---

# Validation Execution Strategy

Validation executes using:

```txt
250ms debounce
```

---

## Why Debouncing Matters

The debounce balances:

- responsive UX
- parser stability
- large-schema performance
- predictable CPU usage

without reparsing on every keystroke.

---

# Validation Pipeline

```text
Editor Change
      │
      ▼
250ms debounce
      │
      ▼
yaml.parse()
      │
      ├── Parse Error
      │        ▼
      │   Diagnostics
      │
      ▼
Zod Validation
      │
      ├── Validation Failure
      │         ▼
      │    Diagnostics
      │
      ▼
preprocessComponent()
      │
      ├── Runtime Warning
      │         ▼
      │    Warning Diagnostic
      │
      ▼
Valid Runtime Schema
```

---

# Error Severity Levels

The validator distinguishes between:

| Severity | Meaning |
|---|---|
| Error | Schema cannot safely render |
| Warning | Runtime assumptions may fail |
| Info | Non-blocking guidance |

---

# Validation Design Goals

The validation system is intentionally designed to:

- prevent editor crashes
- preserve typing responsiveness
- mirror runtime behavior closely
- remain future-compatible
- support extensibility
- avoid false positives

---

# Accessibility Considerations

Validation feedback remains accessible through:

- keyboard navigation
- focusable diagnostics
- visible severity indicators
- screen-reader compatible annotations where supported

Accessibility is treated as part of the validation system itself rather than separate UI polish.

---

# Future Extensibility

The validation layer is intentionally structured for future enhancements such as:

- provider-aware validation
- CEL diagnostics
- semantic lint rules
- topology dependency checks
- schema migration warnings

without requiring architectural rewrites.

---

# 6.4 Mock-Provider Infrastructure

---

## Goals

The mock-provider system allows plugin authors to prototype provider-backed fields without requiring backend APIs.

This is critical for:

- offline development
- isolated experimentation
- downstream forks
- rapid schema iteration
- local preview workflows

---

# New Files

---

## Mock Extraction Layer

```txt
pages/plugin-developer/mock-providers/extract-mock-data.ts
```

Responsible for discovering inline:

```yaml
mockData:
```

definitions inside schemas.

---

## Mock Registration Layer

```txt
pages/plugin-developer/mock-providers/register-mocks.ts
```

Responsible for:

- provider registration
- refresh cycles
- stale cleanup
- ownership tracking

---

## Mock Provider Panel

```txt
pages/plugin-developer/mock-providers/mock-providers-panel.tsx
```

Provides runtime visibility into currently active mock providers.

---

# Mock Infrastructure Test Files

```txt
pages/plugin-developer/mock-providers/*.test.ts(x)
```

---

## Mock Infrastructure Coverage

Coverage includes:

- provider extraction
- lifecycle refresh behavior
- stale cleanup
- collision prevention
- ownership tracking
- registry synchronization

---

# Inline mockData Format

Example:

```yaml
dataSource:
  provider: storage-classes

  mockData:
    - label: fast-storage
      value: fast-storage
```

---

# Why Inline Mocks Were Chosen

Inline mocks make schemas:

- portable
- reproducible
- self-contained
- collaboration-friendly

A single YAML file fully describes:

- UI schema
- provider behavior
- preview state

without requiring external fixtures.

---

# Extraction Pipeline

The parser walks the schema tree and extracts:

```ts
{
  provider,
  options
}
```

pairs for temporary provider registration.

---

# Registry Refresh Lifecycle

```text
Parse YAML
    │
    ▼
Extract mockData
    │
    ▼
Diff owned providers
    │
    ▼
Unregister stale mocks
    │
    ▼
Register active mocks
```

This guarantees preview behavior always reflects the current editor state.

---

# Required Additive Runtime Change

A small additive helper is introduced inside:

```txt
api-providers/registry.ts
```

---

## Proposed Helper

```ts
unregister(key: string): boolean {
  return this.entries.delete(key);
}
```

---

# Why unregister() Is Necessary

The current registry already protects against duplicate registration:

```ts
if (this.entries.has(key) && !import.meta.hot)
```

Without unregister support:

- stale providers accumulate
- edits require reloads
- dynamic mock refresh becomes unreliable

---

# Why the Change Remains Safe

The helper is:

- additive
- isolated
- backward-compatible

Existing production provider behavior remains unchanged.

Only playground-owned providers use unregister semantics.

---

# Ownership Tracking

The registrar tracks owned providers explicitly:

```ts
const ownedKeys = new Set<string>();
```

---

# Why Ownership Tracking Matters

Ownership tracking prevents:

- production-provider shadowing
- registry corruption
- stale provider leakage
- accidental collisions

during live editing.

---

# Collision Prevention

Mock providers never silently override production providers.

The registrar checks:

```ts
providerRegistry.has(key)
```

before registration.

---

# Collision Behavior

If a conflict exists:

- the production provider remains authoritative
- the mock is ignored
- a warning diagnostic is surfaced

This preserves runtime safety.

---

# Mock Refresh Semantics

Editing mock definitions dynamically refreshes preview state without requiring page reloads.

The lifecycle supports:

- add
- remove
- update
- rename

operations safely.

---

# Why Dynamic Refresh Matters

Without live refresh behavior:

- previews drift from editor state
- stale providers remain active
- iteration becomes unreliable

The playground is intentionally designed for fast schema experimentation.

---

# Runtime Reuse Philosophy

The proposal intentionally reuses the real:

```ts
providerRegistry
```

instead of creating a separate playground-only abstraction.

Benefits include:

- runtime parity
- lower maintenance burden
- earlier bug discovery
- fewer duplicated code paths

---

# Accessibility Considerations

The mock-provider panel supports:

- keyboard navigation
- focus visibility
- accessible labels
- readable provider state presentation

Accessibility remains part of the baseline implementation quality bar.

# 6.5 Runtime Preview & Output Panel

---

## Goals

The preview pane validates schemas against the real production runtime.

The implementation intentionally avoids introducing any parallel rendering abstraction.

Everything renders through the same runtime infrastructure already used in production.

---

# New Files

---

## Form Preview

```txt
pages/plugin-developer/preview/form-preview.tsx
```

Responsible for:

- runtime rendering
- topology switching
- form synchronization
- preview lifecycle handling

---

## Output Panel

```txt
pages/plugin-developer/preview/output-panel.tsx
```

Responsible for displaying processed submission payloads after runtime postprocessing.

---

# Preview Test Files

```txt
pages/plugin-developer/preview/*.test.tsx
```

---

## Preview Test Coverage

Coverage includes:

- topology switching
- runtime rendering
- processed output generation
- postprocessing correctness
- synchronization behavior
- form state updates

---

# Runtime Rendering Strategy

The preview renders through the real production:

```tsx
<UIGenerator />
```

using:

- `useFormEngine`
- `FormMode.New`
- production preprocessing
- production postprocessing
- topology-aware rendering

---

# Why Runtime Reuse Matters

The playground must validate against the exact runtime plugin authors eventually ship against.

Duplicating rendering logic would introduce:

- behavior drift
- validation inconsistencies
- maintenance overhead
- debugging ambiguity

The proposal intentionally keeps the playground as a thin developer layer around the production runtime.

---

# Runtime Flow

```text
Parsed Schema
      │
      ▼
useFormEngine
      │
      ▼
UIGenerator
      │
      ▼
Rendered Form
      │
      ▼
postprocessSchemaData()
      │
      ▼
Output Panel
```

---

# Form Preview Responsibilities

The preview pane supports:

- live rerendering
- topology navigation
- runtime validation
- provider-backed fields
- conditional rendering
- state synchronization

without introducing playground-specific rendering behavior.

---

# Topology Switching

Multi-topology schemas are supported directly through the runtime flow.

---

## Why Topology Switching Matters

Real provider schemas may contain:

- standalone deployments
- HA deployments
- distributed topologies
- restore flows

The playground must support the same navigation structure production uses.

---

# Preview Synchronization Behavior

The preview rerenders automatically whenever:

- schema structure changes
- mock providers change
- topology changes
- validation succeeds

---

# Failure Handling Philosophy

Preview failures should never crash the application shell.

Unexpected runtime failures are isolated through existing app-level boundaries.

---

# Error Handling Strategy

User-facing validation failures remain inline inside the editor.

Unexpected runtime exceptions bubble into the existing:

```txt
App.tsx
```

error boundary infrastructure.

No playground-specific crash system is introduced.

---

# Why Existing Error Boundaries Are Reused

Reusing the existing application-level error system preserves:

- runtime consistency
- maintainability
- architectural simplicity
- predictable debugging behavior

---

# Output Panel

The output panel renders processed payloads after:

```ts
postprocessSchemaData()
```

---

# Why the Output Panel Exists

Plugin authors often need to inspect:

- transformed payloads
- normalized values
- expanded paths
- stripped fields
- submit-ready JSON

The preview alone does not expose this runtime behavior clearly.

---

# Output Rendering Strategy

The output panel uses a read-only CodeMirror instance with JSON highlighting.

Benefits:

- syntax highlighting
- formatting consistency
- reusable editor infrastructure
- copy-friendly rendering

---

# Output Flow

```text
Form State
    │
    ▼
postprocessSchemaData()
    │
    ▼
JSON Serialization
    │
    ▼
Read-Only Output Editor
```

---

# Why postprocessSchemaData() Is Important

The runtime already performs transformations around:

- dynamic paths
- payload shaping
- empty-value stripping
- normalization

The playground must expose these transformations to plugin authors.

---

# Preview Performance Strategy

The preview intentionally rebuilds only when validation succeeds.

This prevents unnecessary runtime churn during malformed edits.

---

# Why This Matters

Without guarded rebuild behavior:

- invalid schemas repeatedly crash preview state
- typing responsiveness suffers
- runtime synchronization becomes unstable

---

# Accessibility Considerations

The preview pane supports:

- keyboard navigation
- focus visibility
- topology switching accessibility
- readable output rendering
- predictable focus recovery

Accessibility is treated as part of the runtime workflow itself.

---

# Future Extensibility

The preview architecture intentionally leaves room for:

- side-by-side topology comparisons
- schema diff previews
- visual validation overlays
- form snapshot exports
- runtime performance instrumentation

without requiring architectural rewrites.

---

# 6.6 Persistence Layer

---

## Goals

The persistence system prevents plugin authors from losing work and supports iterative schema-development workflows.

The implementation intentionally remains lightweight and fully client-side.

---

# New Files

---

## localStorage Layer

```txt
pages/plugin-developer/persistence/local-store.ts
```

Responsible for:

- serialization
- persistence reads/writes
- migrations
- quota handling
- corruption recovery

---

## Persistence Hook

```txt
pages/plugin-developer/persistence/use-persisted-schemas.ts
```

Provides the runtime persistence interface for the playground.

---

# Persistence Test Files

```txt
local-store.test.ts
```

---

## Persistence Coverage

Coverage includes:

- CRUD operations
- autosave behavior
- corrupted-state recovery
- migration handling
- quota failure behavior

---

# Persistence Philosophy

The persistence system intentionally prioritizes:

- simplicity
- reliability
- offline capability
- zero backend dependency

over cloud synchronization complexity.

---

# localStorage Schema

```ts
type PersistedSchema = {
  name: string;
  yaml: string;
  updatedAt: number;
};

type PersistedState = {
  schemas: PersistedSchema[];
  current: string | null;
  version: 1;
};
```

---

# Why Versioned Persistence Exists

Versioning allows future migration support without breaking older saved schemas.

This becomes important as the playground evolves over time.

---

# Autosave Strategy

Autosave triggers:

```txt
1 second after the last keystroke
```

---

# Why Autosave Uses a Delay

The delay balances:

- user safety
- write frequency
- responsiveness
- browser performance

without flooding localStorage writes during rapid typing.

---

# Persistence Workflow

```text
Editor Change
      │
      ▼
Autosave Debounce
      │
      ▼
Serialize State
      │
      ▼
Persist to localStorage
      │
      ▼
Restore on Reload
```

---

# Supported Persistence Features

The persistence layer supports:

- autosave
- schema creation
- rename
- duplicate
- delete
- restore
- active-schema tracking

---

# Why localStorage Was Chosen First

Advantages include:

- zero backend coordination
- offline support
- immediate usability
- low implementation overhead
- reduced authentication complexity

Future sync systems can layer on later if needed.

---

# Failure Handling

Persistence failures should never destroy in-memory editing state.

---

# Quota Failure Strategy

If storage limits are exceeded:

- persistence fails gracefully
- warning UI appears
- current in-memory state remains intact
- users can clean old schemas manually

---

# Why Graceful Failure Matters

Large schema libraries may eventually exceed browser storage quotas.

The playground must remain usable even when persistence cannot continue.

---

# Corruption Recovery

Malformed or partially corrupted persistence payloads are automatically reset safely.

---

# Recovery Philosophy

The implementation prioritizes:

- editor stability
- predictable recovery
- avoiding fatal startup failures

over preserving potentially corrupted state.

---

# Persistence Design Goals

The persistence system is intentionally designed to remain:

- backend-independent
- portable
- future-compatible
- low-maintenance
- resilient under failure

---

# Accessibility Considerations

Persistence workflows support:

- keyboard-accessible schema switching
- readable schema labels
- focus recovery after save operations
- accessible warning messages

Accessibility remains integrated into workflow behavior rather than added later.

---

# Future Extensibility

The persistence architecture intentionally leaves room for:

- cloud synchronization
- GitHub gist exports
- collaborative editing
- workspace snapshots
- remote schema libraries

without replacing the core persistence model.

# 6.7 Import & Export Workflows

---

## Goals

The import/export system allows plugin authors to:

- share schemas easily
- persist work externally
- reproduce playground sessions
- migrate schemas between environments
- build reusable schema libraries

The implementation intentionally remains lightweight and browser-native.

---

# New Files

---

## Import Workflow

```txt
pages/plugin-developer/io/import-yaml.tsx
```

Responsible for:

- file selection
- YAML loading
- validation integration
- import error handling

---

## Export Workflow

```txt
pages/plugin-developer/io/export-yaml.ts
```

Responsible for:

- schema serialization
- Blob generation
- download triggering
- optional mock stripping

---

# Import Strategy

The import flow uses:

```html
<input type="file" accept=".yaml,.yml">
```

combined with:

```ts
FileReader.readAsText()
```

---

# Import Pipeline

```text
Select File
     │
     ▼
Read File Text
     │
     ▼
YAML Parse
     │
     ▼
Validation Pipeline
     │
     ├── Validation Failure
     │         ▼
     │    Diagnostics
     │
     ▼
Restore Into Editor
```

---

# Why Imports Use the Existing Validation Pipeline

Imported schemas should behave identically to manually edited schemas.

Reusing the same pipeline guarantees:

- consistent diagnostics
- runtime parity
- predictable behavior
- lower maintenance overhead

---

# Supported File Types

The import system supports:

```txt
.yaml
.yml
```

only.

---

# Why YAML Is Standardized Initially

The playground intentionally focuses on:

```txt
YAML-first workflows
```

because the existing V2 plugin ecosystem already uses YAML schemas extensively.

Supporting multiple formats immediately would increase:

- parser complexity
- validation ambiguity
- testing surface area

without clear ecosystem benefit initially.

---

# Import Failure Handling

Malformed imports never crash the application shell.

Validation failures surface as normal diagnostics inside the editor.

---

# Export Strategy

Exports use:

```ts
URL.createObjectURL(
  new Blob([text], {
    type: 'text/yaml',
  })
)
```

combined with a programmatic download trigger.

---

# Export Workflow

```text
Current Schema
      │
      ▼
Optional mockData stripping
      │
      ▼
Serialize YAML
      │
      ▼
Create Blob
      │
      ▼
Trigger Download
```

---

# mockData Stripping

Exports include an optional:

```txt
strip mockData on export
```

toggle.

---

# Why mockData Stripping Exists

Inline mocks are useful during development but may not belong inside production CRDs.

The export toggle helps plugin authors generate:

- playground-oriented exports
- production-oriented exports

from the same schema source.

---

# Default Export Behavior

| Export Mode | mockData Behavior |
|---|---|
| Playground Export | Preserved |
| Production Export | Removed |

---

# Why Export Flexibility Matters

Some workflows benefit from:

- fully reproducible playground files

while others require:

- production-clean provider schemas

The export system intentionally supports both.

---

# YAML Serialization Philosophy

The exporter intentionally preserves:

- readable formatting
- stable indentation
- predictable ordering

where possible.

---

# Why Stable Output Matters

Stable exports improve:

- Git diffs
- reviewability
- collaboration
- debugging
- schema sharing

especially in OSS workflows.

---

# Import/Export Design Goals

The workflow is intentionally designed to remain:

- browser-native
- dependency-light
- portable
- reproducible
- future-compatible

---

# Accessibility Considerations

The import/export workflow supports:

- keyboard-accessible actions
- readable labels
- visible focus states
- accessible warning messaging

Accessibility remains part of the baseline workflow quality.

---

# Future Extensibility

The architecture intentionally leaves room for future additions such as:

- GitHub gist export
- downloadable schema bundles
- cloud synchronization
- copy-to-clipboard workflows
- multi-schema workspaces

without replacing the existing implementation.

---

# 6.8 Legacy POC Removal

---

## Goals

Once functional parity exists, the existing Monaco-based playground proof-of-concept is removed completely.

The repository should expose:

```txt
one officially supported playground implementation
```

rather than maintaining parallel experimental systems.

---

# Removal Targets

The following legacy implementation is removed:

```txt
pages/ui-generator-builder/
```

---

# Route Cleanup

The legacy route entry is removed from:

```txt
router.tsx
```

after the new playground reaches feature parity.

---

# Cleanup Strategy

Additional repository cleanup includes:

```bash
grep -r ui-generator-builder docs/ ui/
```

to locate:

- stale references
- outdated documentation
- obsolete screenshots
- unused imports

---

# Why Immediate Cleanup Matters

Maintaining two competing playground systems introduces:

- contributor confusion
- fragmented testing
- duplicated maintenance
- unclear architectural direction

The repository should communicate a single long-term solution clearly.

---

# Removal Preconditions

The Monaco-based POC is removed only after:

- runtime parity is verified
- CSP validation passes
- preview behavior matches expectations
- import/export workflows exist
- persistence workflows exist
- tests are green

---

# Why the POC Is Not Removed Earlier

Premature removal would risk destabilizing active experimentation during development.

Keeping the POC temporarily available reduces migration risk while the replacement matures.

---

# Migration Philosophy

The migration intentionally follows:

```txt
replace after parity
```

rather than:

```txt
rewrite in parallel indefinitely
```

This keeps repository direction clear and maintainable.

---

# Post-Removal Validation

After cleanup:

- all playground routes should resolve correctly
- no Monaco dependencies should remain
- CSP verification should remain green
- all tests should pass
- no stale imports should exist

---

# Dependency Cleanup

Monaco-related dependencies are removed after migration completion.

This includes:

- editor packages
- stale playground utilities
- obsolete split-pane code
- unused helpers

where applicable.

---

# Why Dependency Cleanup Matters

Removing unused dependencies improves:

- bundle clarity
- maintenance overhead
- dependency auditing
- security reviewability

especially in security-sensitive frontend systems.

---

# Documentation Cleanup

All outdated references to:

```txt
/ui-generator-builder
```

are removed from:

- docs
- screenshots
- onboarding instructions
- contributor guides

---

# Final Migration Goal

The final repository state should provide:

- one CSP-safe playground
- one supported editor architecture
- one validation flow
- one runtime preview path

with no ambiguity around long-term direction.

---

# Technical Implementation Summary

The complete implementation intentionally reuses the production runtime wherever possible.

---

# Core Architectural Principles

The playground is designed around:

- runtime reuse
- additive architecture
- CSP safety
- minimal abstraction duplication
- incremental delivery
- strong test coverage
- future extensibility

---

# Key Runtime Systems Reused

The implementation directly reuses:

- `UIGenerator`
- `useFormEngine`
- `preprocessComponent`
- `postprocessSchemaData`
- `providerRegistry`
- existing CSP nonce flow

rather than introducing parallel infrastructure.

---

# Why Runtime Reuse Is Critical

Runtime reuse guarantees:

- production parity
- lower maintenance cost
- fewer hidden inconsistencies
- easier debugging
- safer long-term evolution

---

# Security Summary

The proposal intentionally preserves the existing production CSP unchanged.

No additional permissions are introduced.

The implementation avoids:

- `unsafe-eval`
- blob workers
- iframe workarounds
- runtime code generation
- CSP relaxation

---

# Architectural Validation Already Completed

The local proof-of-concept already validated:

| Capability | Status |
|---|---|
| CodeMirror under strict CSP | Confirmed |
| Nonce propagation | Confirmed |
| Split-pane rendering | Confirmed |
| Route integration | Confirmed |
| Inline validation | Confirmed |
| Runtime compatibility | Confirmed |
| Zero CSP violations | Confirmed |
| Test coverage foundation | Confirmed |

---

# Expected Final Result

By the end of implementation, the playground should provide:

- CSP-safe YAML editing
- live UIGenerator preview
- inline validation diagnostics
- mock provider support
- import/export workflows
- autosave persistence
- accessibility-aware interactions
- production-safe E2E validation

while preserving OpenEverest’s existing runtime and security architecture unchanged.

# 7. 12-Week Timeline

---

## Timeline Philosophy

The implementation is intentionally structured around:

- small reviewable PRs
- additive architecture
- runtime reuse
- continuous mentor feedback
- early CSP validation
- incremental delivery

The roadmap prioritizes:

1. security-critical infrastructure first
2. runtime integration second
3. developer workflow tooling third
4. polish and cleanup last

---

# High-Level Timeline

| Week | Focus | Deliverables |
|---|---|---|
| 1–2 | Environment + infrastructure | Runtime study, `/plugin-developer` route, split-pane shell |
| 3–4 | Editor foundation | CodeMirror integration, CSP nonce wiring |
| 5 | Validation pipeline | YAML parsing, Zod validation, inline diagnostics |
| 6 | Runtime preview | `UIGenerator` integration, topology switching, output panel |
| 7 | Mock providers | Dynamic provider registration + refresh lifecycle |
| 8 | Stabilization | Rebases, cleanup, review feedback |
| 9 | Persistence workflows | Autosave, schema management, import/export |
| 10 | Testing + accessibility | Expanded Vitest coverage, accessibility hardening |
| 11 | E2E + migration cleanup | Playwright CSP tests, legacy Monaco POC removal |
| 12 | Documentation + wrap-up | Final docs, troubleshooting, release cleanup |

---

# PR Strategy

The implementation intentionally follows:

```txt
small independently reviewable PRs
```

rather than one large integration branch.

Expected PR flow:

| PR | Scope |
|---|---|
| PR 1 | Route shell + split-pane |
| PR 2 | Documentation skeleton |
| PR 3 | CodeMirror + CSP integration |
| PR 4 | Validation pipeline |
| PR 5 | Runtime preview |
| PR 6 | Mock-provider infrastructure |
| PR 7 | Stabilization/refactors |
| PR 8 | Persistence + import/export |
| PR 9 | Testing + accessibility |
| PR 10 | E2E + legacy POC removal |
| PR 11 | Final documentation + cleanup |

---

# Timeline Risk Mitigation

The roadmap intentionally includes:

- an explicit stabilization/buffer week
- continuous rebasing against V2 churn
- incremental documentation updates
- production-preview CSP validation early
- isolated implementation slices

This reduces the risk of late-stage architectural rewrites or merge conflicts.

---

# Expected Final Deliverables

By the end of the mentorship, the playground should provide:

- CSP-safe YAML editing
- live `UIGenerator` preview
- inline schema diagnostics
- mock-provider support
- import/export workflows
- local persistence
- accessibility-aware interactions
- production-safe E2E verification

while preserving OpenEverest’s existing runtime and CSP architecture unchanged.

# 8. Testing & Validation Strategy

---

## Testing Philosophy

The Plugin Developer Playground interacts directly with:

- runtime schema rendering
- provider registries
- validation pipelines
- CSP enforcement
- YAML parsing
- runtime preprocessing/postprocessing

Because of this, correctness cannot rely on manual testing alone.

The testing strategy is designed to guarantee:

- runtime safety
- CSP compatibility
- stable editor behavior
- registry lifecycle correctness
- accessibility compliance
- long-term maintainability

---

# Testing Stack

| Layer | Tooling |
|---|---|
| Unit tests | Vitest |
| Component tests | React Testing Library |
| DOM environment | jsdom |
| Mocking | `vi.mock`, `vi.hoisted` |
| E2E | Playwright |
| Coverage | Vitest coverage |

---

# Existing Repository Standards

The repository already establishes strong frontend testing conventions under:

```txt
components/ui-generator/
```

including:

```txt
19 colocated test files
```

The playground intentionally mirrors those existing patterns to remain consistent with the surrounding V2 runtime architecture.

---

# Planned Test Coverage

The playground targets:

- editor behavior
- validation correctness
- persistence workflows
- runtime preview integration
- mock-provider lifecycle handling
- CSP enforcement
- accessibility interactions

---

# Planned Test Files

| Test File | Primary Coverage |
|---|---|
| `code-editor.test.tsx` | Editor lifecycle + nonce propagation |
| `extensions.test.ts` | Extension configuration |
| `yaml-parser.test.ts` | YAML parsing behavior |
| `topology-ui-schemas.zod.test.ts` | Structural validation |
| `to-diagnostics.test.ts` | Diagnostic conversion |
| `register-mocks.test.ts` | Provider lifecycle behavior |
| `local-store.test.ts` | Persistence workflows |
| `form-preview.test.tsx` | Runtime rendering integration |
| `output-panel.test.tsx` | Postprocessing correctness |
| `split-pane.test.tsx` | Resize behavior + cleanup |

---

# CodeMirror Validation

The editor is the most security-sensitive frontend component in the proposal.

Tests validate:

- nonce propagation
- diagnostics rendering
- update synchronization
- cleanup behavior
- keyboard accessibility
- CSP-safe style injection

---

# YAML & Validation Testing

Validation coverage includes:

- malformed YAML
- invalid topology structures
- partial schema recovery
- line-aware diagnostics
- runtime smoke validation

Real V2 schema fixtures are validated directly to preserve runtime parity.

---

# Mock-Provider Testing

The mock-provider infrastructure validates:

- provider extraction
- refresh behavior
- stale cleanup
- collision prevention
- ownership tracking

The tests intentionally use the real:

```ts
providerRegistry
```

instead of mocked replacements wherever possible.

---

# Persistence Testing

Persistence coverage includes:

- autosave behavior
- CRUD operations
- corrupted-state recovery
- quota failure handling
- migration compatibility

localStorage behavior is simulated using:

```ts
vi.stubGlobal('localStorage', ...)
```

---

# Playwright E2E Strategy

Browser-level testing is mandatory because:

- jsdom does not enforce CSP
- Vite dev mode bypasses the production CSP
- CodeMirror behavior differs in real browsers

The E2E suite validates the playground against:

```bash
pnpm preview
```

instead of:

```bash
pnpm dev
```

---

# Critical E2E Flows

Primary E2E validation includes:

1. open `/plugin-developer`
2. paste valid YAML
3. verify preview rendering
4. introduce validation errors
5. verify diagnostics appear
6. reload page
7. verify persistence restoration

---

# CSP Verification

A dedicated CSP-focused E2E test validates:

- no `eval()` usage
- no CSP console violations
- editor functionality under production CSP
- runtime rendering stability

---

# Coverage Goals

Coverage thresholds apply specifically to new playground modules.

---

## Target Thresholds

```txt
Lines: >80%
Branches: >75%
```

Scoped to:

```txt
pages/plugin-developer/**
components/code-editor/**
```

---

# Accessibility Validation

Accessibility testing includes:

- keyboard-only navigation
- visible focus indicators
- lint navigation behavior
- accessible labels
- predictable focus recovery

Accessibility assertions are integrated throughout the test suite rather than isolated separately.

---

# Manual Validation Checklist

Automated testing is supplemented with manual validation during every implementation slice.

---

## Editor Validation

- syntax highlighting works
- diagnostics appear correctly
- keyboard navigation functions
- no CSP violations occur

---

## Runtime Validation

- real V2 schemas render correctly
- mock providers populate fields correctly
- topology switching behaves correctly
- processed output matches runtime expectations

---

## Persistence Validation

- autosave restores successfully
- import/export round-trips correctly
- schema switching remains stable

---

# Production Readiness Criteria

The implementation is considered production-ready when:

- all tests pass consistently
- CSP E2E passes against production builds
- malformed schemas never crash the UI
- real V2 schemas render correctly
- mock providers behave predictably
- accessibility interactions work correctly
- no backend CSP changes are required

# 9. Documentation Plan

---

## Documentation Goals

The playground documentation is intended to help:

- plugin authors
- future contributors
- maintainers
- downstream providers

quickly understand and extend the system safely.

The focus is on:

- onboarding speed
- architectural clarity
- CSP understanding
- reproducibility
- maintainability

---

# Primary Documentation File

Main documentation:

```txt
docs/ui/ui-generator/playground.md
```

This becomes the central reference for the playground subsystem.

---

# Planned Documentation Sections

## 1. Overview

- what the playground is
- why it exists
- how it fits into the V2 plugin ecosystem

---

## 2. Why Monaco Was Replaced

Covers:

- strict CSP constraints
- Monaco limitations
- CodeMirror 6 rationale
- security trade-offs

---

## 3. CSP Architecture

Documents:

- nonce propagation
- App.tsx integration
- CodeMirror nonce handling
- production-preview requirements

---

## 4. Playground Workflow

Step-by-step usage flow:

1. Open `/plugin-developer`
2. Paste YAML
3. Preview live UI
4. Add mock providers
5. Save/export schemas

---

## 5. Validation System

Explains:

- YAML parsing
- Zod validation
- diagnostics behavior
- preprocess smoke validation

---

## 6. Mock Providers

Documents:

- `mockData:` usage
- provider lifecycle
- collision prevention

---

## 7. Persistence & Import/Export

Documents:

- autosave
- localStorage
- `.yaml` import/export
- quota handling

---

## 8. Testing & CSP Verification

Documents:

- Vitest workflows
- Playwright CSP testing
- production preview requirements

---

## 9. Troubleshooting

Examples:

| Problem | Resolution |
|---|---|
| CSP violations | Verify nonce propagation |
| Preview not rendering | Check topology structure |
| Mock provider missing | Verify provider key uniqueness |

---

# Planned Diagrams

Lightweight diagrams will document:

## CSP Flow

```txt
Go Middleware
    ↓
CSP Nonce
    ↓
App.tsx
    ↓
CodeMirror
```

---

## Runtime Flow

```txt
YAML Editor
    ↓
Validation
    ↓
UIGenerator
    ↓
Preview
```

---

# Additional Documentation Updates

The proposal also updates:

```txt
CONTRIBUTING.md
```

with:

- playground test instructions
- CSP verification workflow
- production preview guidance

---

# Documentation Strategy

Documentation updates will happen incrementally throughout development instead of only at the end of the mentorship.

This helps prevent:

- stale docs
- onboarding gaps
- architectural drift

---

# Success Criteria

Documentation is considered successful when:

- contributors can use the playground independently
- CSP behavior is clearly understood
- playground workflows are reproducible
- future contributors can extend the system safely

# 10. Risk Analysis

---

# Risk Management Philosophy

The project touches several sensitive areas simultaneously:

- CSP enforcement
- runtime schema rendering
- provider registries
- editor infrastructure
- active V2 branch development

To reduce implementation risk, the proposal prioritizes:

- small reviewable PRs
- additive architecture
- production runtime reuse
- early maintainer feedback
- high test coverage
- continuous rebasing

---

# Risk Summary

| Risk | Impact | Mitigation |
|---|---|---|
| V2 branch instability | High | Small PRs + regular rebases |
| Overly strict validation | Medium | Loose nested validation |
| CSP issues hidden in Vite dev mode | Medium | Production-preview E2E testing |
| Mock provider collisions | Medium | Ownership tracking + guarded registry updates |
| Academic workload spikes | Medium | Buffer week + scope prioritization |
| localStorage quota limits | Low | Graceful fallback handling |

---

# Key Technical Risks

---

## 1. V2 Branch Instability

The `v2` branch is still evolving rapidly, especially around:

- UIGenerator
- routing
- form-engine APIs
- provider registries

### Mitigation

- small incremental PRs
- weekly rebases
- isolated implementation slices
- dedicated stabilization week

---

## 2. Overly Strict Validation

`TopologyUISchemas` intentionally includes flexible extension points.

Aggressive validation could reject valid provider-specific schemas.

### Mitigation

The proposal validates:

- top-level structure strictly
- nested component definitions loosely

using:

```ts
z.unknown()
```

for extensibility.

A preprocess smoke pass also catches runtime issues without blocking editing.

---

## 3. CSP Verification Risk

Vite dev mode does not enforce the real production CSP.

This can hide runtime CSP issues during development.

### Mitigation

All CSP verification runs against:

```bash
pnpm preview
```

with dedicated Playwright tests checking for:

- CSP violations
- `eval()` usage
- editor stability

---

## 4. Mock Provider Collisions

Dynamic mock providers must never override production providers accidentally.

### Mitigation

The implementation includes:

- ownership tracking
- collision detection
- guarded registration
- automatic cleanup of stale mocks

---

## 5. Editor Performance on Large Schemas

Large YAML files could affect responsiveness.

### Mitigation

- debounced validation
- lightweight editor extensions
- maximum file-size guard
- warning diagnostics for oversized files

---

## 6. localStorage Limits

Large schema collections may exceed browser storage limits.

### Mitigation

- graceful warning UI
- safe fallback behavior
- schema cleanup guidance

---

# Process Risks

---

## Timezone Coordination

The mentorship involves IST ↔ EU timezone coordination.

### Mitigation

- fixed weekly syncs
- async GitHub updates
- draft PR workflows
- early blocker communication

---

## Academic Workload

Unexpected academic workload spikes are possible during the mentorship.

### Mitigation

If required, lower-priority polish work is deferred before core functionality.

Priority order:

1. CSP-safe editor
2. Runtime preview
3. Validation
4. Mock providers
5. Persistence
6. UX polish

---

# Long-Term Maintainability Risks

---

## Playground Diverging from Production Runtime

A separate rendering implementation could drift over time.

### Mitigation

The playground directly reuses:

- `UIGenerator`
- `useFormEngine`
- preprocess/postprocess pipelines
- provider registries

instead of duplicating runtime logic.

---

## Future Contributor Confusion

Future contributors may not understand:

- why Monaco was removed
- why CSP testing is special
- why validation is intentionally conservative

### Mitigation

Architectural reasoning is documented explicitly in:

```txt
playground.md
```

and GitHub discussions.

---

# Overall Risk Assessment

| Area | Risk Level |
|---|---|
| CSP compatibility | Medium |
| Runtime integration | Medium |
| Validation design | Medium |
| Schedule management | Low |
| Long-term maintainability | Low |

---

# Why the Overall Risk Is Acceptable

The project risk remains manageable because:

- the architecture is additive
- the CSP policy remains unchanged
- CodeMirror 6 is already CSP-safe
- the runtime is reused instead of rewritten
- the implementation is split into small reviewable slices
- testing and stabilization are built into the timeline

---

# Success Criteria

Risk management is considered successful when:

- no CSP policy changes are required
- runtime behavior matches production
- rebases remain manageable
- validation stays flexible
- the editor remains performant
- future contributors can understand the architecture easily

# 11. Community Collaboration Plan

---

# Collaboration Philosophy

The playground touches multiple sensitive parts of the OpenEverest frontend architecture:

- UIGenerator
- provider registries
- schema validation
- CSP/security
- developer tooling

Because of this, successful implementation depends heavily on communication quality and reviewability.

The collaboration approach focuses on:

- small reviewable PRs
- early architectural feedback
- public discussions
- maintainability
- transparent progress updates

---

# Communication Strategy

## Weekly Mentor Syncs

A recurring weekly sync is planned for:

```txt
14:00 IST / 09:30 CET
```

Focus areas:

- review feedback
- blockers
- architecture discussions
- rebasing concerns
- implementation planning

---

## Async Progress Updates

Two structured GitHub updates per week:

| Day | Purpose |
|---|---|
| Monday | Weekly goals + planned work |
| Thursday | Progress, blockers, PR updates |

These updates remain public for contributor transparency.

---

# Communication Channels

| Channel | Purpose |
|---|---|
| GitHub Issues | Architecture + progress |
| GitHub PRs | Code review |
| CNCF Slack | Quick discussions |
| Weekly Calls | High-bandwidth feedback |

---

# Draft PR Workflow

All major implementation slices begin as:

```txt
Draft PRs
```

until:

- CI passes
- tests are complete
- screenshots are attached
- scope is stabilized

This allows maintainers to guide architecture early before implementation hardens.

---

# PR Structure Standards

Each PR will include:

- linked issue
- implementation summary
- architectural rationale
- screenshots/GIFs
- testing notes
- accessibility notes
- CSP considerations (when relevant)

---

# Review Philosophy

The proposal intentionally prioritizes:

- small focused commits
- minimal unrelated refactors
- quick review responses
- explicit architectural reasoning

The goal is to reduce reviewer overhead and keep review cycles fast.

---

# Repository Standards

All contributions follow repository conventions:

- DCO sign-offs
- Apache 2.0 headers
- colocated tests
- screenshot requirements
- accessibility-aware UI changes

---

# Architectural Discussion Strategy

Major decisions will follow an RFC-style discussion approach.

Examples include:

- Monaco vs CodeMirror
- validation strictness
- mock provider lifecycle
- persistence design

This keeps architectural reasoning discoverable for future contributors.

---

# Long-Term Contribution Intent

The goal is not only to complete one mentorship project.

The intention is to continue contributing in:

```txt
area/ui
```

particularly around:

- schema tooling
- developer experience
- frontend testing
- plugin infrastructure
- CSP-safe frontend systems

---

# Knowledge Sharing Goals

The implementation process aims to generate reusable knowledge through:

- public design discussions
- documentation
- testing examples
- CSP implementation notes
- architectural rationale

This is especially valuable because Monaco vs CSP issues are common across modern frontend platforms.

---

# Success Criteria

The collaboration effort is considered successful when:

- review cycles remain healthy
- maintainers can follow implementation history easily
- architectural decisions remain publicly documented
- future contributors can understand the playground architecture quickly
- the playground becomes maintainable long-term infrastructure instead of a one-off feature

# 12. Why I'm a Strong Candidate

---

# Project Fit

This project combines several areas that align closely with my experience and interests:

- React + TypeScript frontend architecture
- schema-driven UI systems
- developer tooling
- CSP-safe frontend engineering
- runtime validation workflows
- test-heavy frontend development
- OSS collaboration

The proposal was built after studying the actual V2 implementation and runtime architecture rather than relying only on the issue description.

---

# Relevant Preparation Work

Before writing this proposal, I completed significant investigation work against the OpenEverest V2 branch, including:

- studying the V2 plugin architecture
- reproducing the Monaco CSP issue locally
- tracing the nonce propagation flow
- analyzing the provider registry lifecycle
- reviewing existing frontend testing patterns
- evaluating CodeMirror feasibility under strict CSP
- studying maintainer review patterns and repository conventions

---

# Repository Areas Studied

The following systems were analyzed directly:

## V2 Runtime

- `ui-generator.tsx`
- `ui-generator.types.ts`
- `use-form-engine.ts`
- `preprocess-schema.ts`
- `postprocess-schema.ts`

---

## Provider Registry System

- `api-providers/registry.ts`
- `api-providers/types.ts`

---

## Existing Playground POC

- `pages/ui-generator-builder/`

including:

- Monaco integration
- split-pane implementation
- preview flow
- current limitations

---

## CSP & Security Pipeline

- `internal/server/middlewares.go`
- `App.tsx`
- `index.html`

This included tracing:

```txt
Go Middleware
→ CSP Nonce
→ Meta Tag
→ React Runtime
→ Emotion
→ CodeMirror
```

---

# Local Validation Work

The proposal is based on direct local validation against the V2 branch.

Executed locally:

```bash
git checkout v2
pnpm install
pnpm dev
```

This allowed validation of:

- current playground behavior
- runtime rendering flow
- CSP constraints
- provider registry interactions
- editor integration assumptions

---

# Existing PoC Work

A CSP-safe CodeMirror 6 proof-of-concept was already explored locally during proposal preparation.

The PoC demonstrated:

- nonce propagation
- YAML editing
- split-pane rendering
- validation plumbing
- route integration
- test coverage
- zero CSP violations

This substantially reduced architectural uncertainty before implementation planning.

---

# Testing-Oriented Development

The proposal intentionally emphasizes:

- colocated tests
- runtime-oriented validation
- accessibility testing
- Playwright E2E coverage
- CSP-specific verification

The PoC already includes:

```txt
31 tests across 6 test files
```

covering editor behavior, validation, layout, and nonce handling.

---

# Security Awareness

A major focus of the proposal is preserving OpenEverest’s strict CSP model.

The implementation deliberately avoids:

- `unsafe-eval`
- worker-based hacks
- iframe workarounds
- CSP relaxation
- runtime code generation

Instead, the solution reuses the existing nonce architecture already present in production.

---

# OSS Collaboration Approach

The implementation strategy intentionally follows an RFC-style workflow:

- ask for architectural feedback early
- keep PRs reviewable
- avoid large surprise changes
- document reasoning publicly
- align with repository conventions

The proposal prioritizes:

- maintainability
- reviewability
- documentation quality
- testing discipline
- long-term sustainability

rather than maximizing feature count.

---

# Honest Gap Analysis

There are areas where experience is still developing, including:

- deeper CodeMirror extension internals
- Lezer parser internals
- production CSP-focused Playwright workflows

However, these are implementation-depth gaps rather than architectural unknowns.

The critical architectural assumptions have already been validated through local investigation and PoC work.

---

# Long-Term Contributor Intent

The goal is not simply to complete one mentorship project.

I intend to continue contributing in:

```txt
area/ui
```

particularly around:

- plugin tooling
- developer experience
- frontend architecture
- testing infrastructure
- schema-driven systems
- CSP-safe frontend tooling

---

# Contribution Philosophy

The intended contribution style throughout the mentorship is:

- architecture-first
- security-conscious
- test-heavy
- documentation-oriented
- incrementally deliverable
- review-friendly

The focus is on building maintainable infrastructure that future contributors can extend safely.

---

# Desired Mentorship Outcome

The mentorship is considered successful if:

- the playground becomes stable long-term infrastructure
- maintainers trust future contributions independently
- future contributors can extend the system confidently
- the CSP-safe editor approach becomes institutionalized knowledge inside the project
- the V2 plugin ecosystem becomes easier to contribute to overall

# 13. Prior Contributions to This Project

---

# Contribution Status

At the time of writing this proposal, formal upstream contributions are still in progress, but significant repository investigation, architectural analysis, and local validation work has already been completed against the OpenEverest codebase.

Rather than overstating contribution history, this section documents the concrete preparation and contribution work already completed before the mentorship application.

---

# Active Contribution Work

## UI Contribution Preparation

An initial UI-focused contribution is currently being prepared/reviewed to align with repository expectations and workflows.

The goal of this contribution is to establish familiarity with:

- repository conventions
- testing expectations
- DCO workflow
- review iteration patterns
- screenshot requirements
- maintainer feedback style

The contribution intentionally follows the repository’s preferred style:

- small focused diff
- tests included
- no unrelated refactors
- reviewable scope

---

# Playground Architecture Investigation

A major portion of the preparation effort focused specifically on Issue `#2059` and the V2 plugin runtime architecture.

This investigation included:

- CSP analysis
- runtime architecture tracing
- provider registry analysis
- validation pipeline study
- existing POC analysis
- dependency evaluation
- implementation risk analysis

---

# Repository Areas Studied

The following systems were analyzed directly in detail.

---

## V2 Plugin Runtime

Studied files:

- `ui-generator.tsx`
- `ui-generator.types.ts`
- `use-form-engine.ts`
- `use-topology.ts`

Focus areas:

- runtime rendering flow
- topology switching
- schema-driven rendering
- form lifecycle behavior

---

## Schema Processing Pipeline

Studied files:

- `preprocess-schema.ts`
- `postprocess-schema.ts`

Focus areas:

- normalization flow
- runtime transformations
- validation integration points
- payload shaping

---

## Provider Registry System

Studied files:

- `api-providers/registry.ts`
- `api-providers/types.ts`

Focus areas:

- provider registration lifecycle
- duplicate registration handling
- runtime provider resolution
- mock-provider feasibility

---

## Existing Playground POC

Studied:

```txt
pages/ui-generator-builder/
```

including:

- Monaco integration
- split-pane architecture
- preview rendering
- current limitations
- CSP incompatibilities

---

# CSP & Security Investigation

A detailed investigation of the existing CSP architecture was completed.

Studied files:

- `internal/server/middlewares.go`
- `App.tsx`
- `index.html`

This included tracing:

```txt
Go Middleware
→ CSP Nonce
→ Meta Tag
→ Emotion
→ React Runtime
→ CodeMirror
```

The key architectural finding was validating that:

```txt
CodeMirror 6 can reuse the existing nonce flow
without requiring CSP policy changes
```

---

# Local Environment Validation

The repository was built and tested locally against the `v2` branch.

Executed locally:

```bash
git checkout v2
pnpm install
pnpm dev
```

This allowed validation of:

- current POC behavior
- runtime rendering flow
- route structure
- provider interactions
- CSP limitations

---

# Existing PoC Prototype Work

A CSP-safe CodeMirror 6 prototype was also explored locally during proposal preparation.

The prototype currently demonstrates:

- YAML editing
- syntax highlighting
- nonce propagation
- split-pane rendering
- validation plumbing
- route integration
- initial test coverage

This work was intentionally scoped as a feasibility prototype rather than a production implementation.

---

# Testing Preparation

The repository’s testing conventions were studied extensively before planning implementation work.

Areas analyzed included:

- colocated test structure
- QueryClient wrappers
- React Testing Library patterns
- Vitest mocking approaches
- Playwright E2E setup

Reference file studied heavily:

```txt
BackupStoragesInput.test.tsx
```

which established the expected frontend testing quality bar.

---

# Maintainer & Review Process Research

Preparation also included studying:

- merged PRs
- review comment patterns
- architectural feedback trends
- UI review expectations
- contribution conventions

Key lessons identified:

- avoid large monolithic PRs
- prioritize tests
- use Draft PRs for architectural work
- keep discussions public
- avoid speculative refactors

These findings directly shaped the implementation strategy in this proposal.

---

# Contribution Philosophy

The contribution approach intentionally emphasizes:

- incremental delivery
- architecture-aware implementation
- review-friendly PRs
- testing discipline
- documentation quality
- maintainability

rather than maximizing raw PR count.

---

# Long-Term Contribution Intent

The goal is not only to complete one mentorship project.

The intention is to continue contributing in:

```txt
area/ui
```

particularly around:

- plugin tooling
- frontend infrastructure
- schema-driven systems
- developer tooling
- testing infrastructure
- CSP-safe frontend engineering

---

# Summary

Before formally applying, the following preparation work had already been completed:

- V2 runtime investigation
- local environment validation
- CSP analysis
- provider registry study
- Monaco incompatibility verification
- CodeMirror feasibility validation
- testing convention research
- maintainer workflow analysis
- PoC prototyping
- active upstream contribution preparation

This proposal is therefore grounded in direct repository familiarity rather than speculative implementation assumptions.

# 14. Long-Term Vision Post-Mentorship

---

# Long-Term Goal

The goal of this mentorship is not only to complete a single feature, but to establish a sustainable foundation for plugin-author tooling inside OpenEverest V2.

The playground is intentionally designed as long-term infrastructure rather than a temporary demo environment.

---

# Immediate Post-Mentorship Commitments

After the mentorship period ends, I intend to continue contributing through:

- bug fixes
- documentation updates
- validation improvements
- accessibility refinements
- testing maintenance
- plugin tooling improvements

particularly in:

```txt
area/ui
```

and the V2 plugin ecosystem.

---

# Long-Term Technical Directions

The mentorship scope focuses on a production-grade foundation first, but several future directions naturally extend from that architecture.

---

## 1. Reusable UISchema Editor Package

Once the playground architecture stabilizes, the editor layer could evolve into a reusable package.

Potential direction:

```txt
@openeverest/codemirror-uischema
```

Possible capabilities:

- UISchema-aware linting
- autocomplete
- diagnostics
- topology helpers
- reusable CodeMirror extensions

This would allow reuse across:

- future tooling
- downstream projects
- IDE integrations
- CLI workflows

---

## 2. Richer Autocomplete & Schema Assistance

The current proposal intentionally keeps autocomplete lightweight.

Future improvements could include:

- topology-name completion
- field-type suggestions
- provider-key completion
- inline documentation
- schema snippets
- validation-aware suggestions

using CodeMirror completion sources and runtime metadata.

---

## 3. CLI Validation Parity

One important future direction is shared validation logic between:

- browser playground
- local development
- CI workflows

Potential workflow:

```bash
everestctl plugin lint schema.yaml
```

This would allow plugin authors to validate schemas in CI using the same logic as the browser playground.

---

## 4. Interactive Documentation Examples

The playground could eventually integrate directly into documentation workflows.

Possible future capabilities:

- live editable examples
- embedded playground demos
- interactive tutorials
- executable schema documentation

inside:

```txt
docs/ui/ui-generator/
```

---

## 5. Advanced Validation & Linting

The mentorship implementation intentionally keeps validation conservative.

Future improvements may include:

- richer CEL diagnostics
- semantic validation
- dependency analysis
- deprecation warnings
- migration guidance
- provider-aware linting

while still preserving schema flexibility.

---

## 6. Accessibility Improvements

Accessibility should continue evolving after the mentorship.

Potential future improvements:

- improved screen-reader behavior
- better keyboard workflows
- accessibility-focused E2E coverage
- reduced-motion support
- high-contrast themes

---

# Ecosystem Vision

The broader goal is to help OpenEverest evolve toward:

```txt
a fully self-service plugin ecosystem
```

where plugin authors can:

- develop schemas locally
- validate instantly
- preview safely
- debug efficiently
- contribute without Kubernetes-heavy workflows

This reduces friction for both contributors and maintainers.

---

# Why the Playground Matters Strategically

The playground is more than a convenience feature.

It can become:

- the primary onboarding surface for plugin authors
- a validation environment
- a documentation platform
- a future tooling foundation

That is why the implementation prioritizes:

- maintainability
- extensibility
- security
- runtime parity
- testing discipline

instead of short-term feature volume.

---

# Long-Term Contribution Philosophy

The intended long-term contribution style remains:

- architecture-aware
- review-friendly
- documentation-oriented
- test-heavy
- security-conscious
- maintainability-focused

The mentorship is viewed as the start of deeper collaboration rather than a one-time contribution cycle.

---

# Desired Outcome After One Year

The ideal state one year after the mentorship would be:

- plugin authors regularly use the playground
- validation logic is reused elsewhere
- onboarding is easier for contributors
- the editor architecture remains maintainable
- future contributors can extend the system safely
- the playground is treated as stable platform infrastructure

rather than an experimental side tool.

---

# Final Commitment

The intention is not to:

```txt
land one feature and disappear
```

but to continue participating in the OpenEverest ecosystem through:

- continued contributions
- tooling improvements
- architectural discussions
- documentation
- issue triage
- helping future contributors

This project is particularly compelling because it combines:

- frontend engineering
- runtime systems
- developer tooling
- security constraints
- OSS collaboration

in a way that rewards long-term investment and thoughtful iteration.

# 15. References

---

# Repository Files

## Core UIGenerator Runtime

- `ui/apps/everest/src/components/ui-generator/ui-generator.tsx`
- `ui/apps/everest/src/components/ui-generator/ui-generator.types.ts`
- `ui/apps/everest/src/components/ui-generator/form-engine/use-form-engine.ts`
- `ui/apps/everest/src/components/ui-generator/hooks/use-topology.ts`

---

## Provider Registry System

- `ui/apps/everest/src/components/ui-generator/api-providers/registry.ts`
- `ui/apps/everest/src/components/ui-generator/api-providers/types.ts`

---

## Schema Processing Pipeline

- `ui/apps/everest/src/components/ui-generator/utils/preprocess/preprocess-schema.ts`
- `ui/apps/everest/src/components/ui-generator/utils/postprocess/postprocess-schema.ts`

---

## Existing Playground POC

- `ui/apps/everest/src/pages/ui-generator-builder/`

Referenced for:

- Monaco integration
- split-pane implementation
- preview architecture
- current limitations

---

## Router & Application Integration

- `ui/apps/everest/src/router/router.tsx`
- `ui/apps/everest/src/router/router-lazy-pages.tsx`
- `ui/apps/everest/src/App.tsx`
- `ui/apps/everest/index.html`

---

## CSP & Security Infrastructure

- `internal/server/middlewares.go`

Referenced for:

- CSP headers
- nonce generation
- security constraints
- production middleware behavior

---

## Existing Testing Patterns

- `ui/apps/everest/src/components/backup-storages-input/BackupStoragesInput.test.tsx`

Used as a reference for:

- colocated tests
- React Testing Library patterns
- QueryClient setup
- frontend testing conventions

---

## Documentation & Contribution Process

- `docs/ui/ui-generator/Readme.md`
- `CONTRIBUTING.md`

Referenced for:

- repository conventions
- documentation structure
- DCO workflow
- review expectations

---

## E2E & CI Infrastructure

- `ui/apps/everest/.e2e/playwright.config.ts`
- `.github/workflows/dev-fe-e2e.yaml`

Referenced for:

- Playwright integration
- production-preview E2E setup
- CI workflow structure

---

# Issues & Discussions

## Primary Tracking Issue

- `openeverest/openeverest#2059`
  - *Plugin Developer Playground: Interactive UI Schema Editor with Live Preview (V2)*

---

## Related Industry Reference

- `grafana/grafana#51047`

Referenced because it documents Monaco Editor challenges under strict CSP environments.

This reinforced the architectural direction toward CodeMirror 6.

---

# External Technical References

## CodeMirror 6

- `https://codemirror.net/docs/guide/`

Referenced for:

- editor architecture
- extension system
- CSP compatibility
- lint integration

---

## @codemirror/lang-yaml

- `https://github.com/codemirror/lang-yaml`

Referenced for:

- YAML grammar support
- syntax highlighting
- parser behavior

---

## @codemirror/lint

- `https://codemirror.net/docs/ref/#lint`

Referenced for:

- diagnostics rendering
- inline linting
- editor validation workflows

---

## Lezer Parser System

- `https://lezer.codemirror.net/`

Referenced for:

- parser architecture
- syntax-tree behavior
- YAML grammar internals

---

## Zod

- `https://zod.dev`

Referenced for:

- runtime schema validation
- TypeScript integration
- validation pipeline design

---

## YAML Package

- `https://eemeli.org/yaml/`

Referenced for:

- YAML parsing
- line/column diagnostics
- parser behavior

---

## React Hook Form

- `https://react-hook-form.com`

Referenced because the V2 runtime already follows react-hook-form patterns.

---

## Material UI (MUI)

- `https://mui.com`

Referenced for:

- existing theme system
- accessibility patterns
- component architecture

---

# CNCF & Mentorship References

## LFX Mentorship Listing

- `https://mentorship.lfx.linuxfoundation.org/project/713b073e-adb4-4d46-95b5-474b8e4c64d9`

---

## CNCF Mentoring Repository

- `https://github.com/cncf/mentoring`

---

# OpenEverest Project References

- `https://openeverest.io`

Used for:

- ecosystem understanding
- platform context
- project positioning

---

# Local Validation Work

The proposal was additionally validated through direct local execution against the `v2` branch.

Executed locally:

```bash
git checkout v2
pnpm install
pnpm dev
```

This was used to validate:

- runtime rendering flow
- CSP limitations
- provider registry behavior
- current playground behavior
- route integration assumptions

---

# Closing Note

Every architectural recommendation in this proposal is grounded in:

- direct repository investigation
- local runtime validation
- existing OpenEverest patterns
- current CSP constraints
- maintainability considerations
- incremental reviewability

with the goal of proposing a solution that integrates naturally into the OpenEverest ecosystem rather than introducing parallel tooling or isolated abstractions.

# Conclusion

The Plugin Developer Playground is designed to solve a real workflow bottleneck in the OpenEverest V2 plugin ecosystem:

```txt
slow, infrastructure-heavy UISchema iteration
```

The proposal introduces a CSP-safe, production-aligned playground that allows plugin authors to:

- edit schemas interactively
- preview forms instantly
- validate structures early
- prototype provider-backed fields locally
- iterate without Kubernetes deployment loops

while preserving OpenEverest’s existing security guarantees and runtime architecture.

---

The implementation intentionally prioritizes:

- runtime reuse over parallel abstractions
- CSP safety over editor feature bloat
- incremental reviewable PRs
- strong automated testing
- long-term maintainability
- contributor onboarding

rather than short-term feature volume.

---

A major focus of the proposal is architectural alignment with the existing V2 ecosystem.

The playground directly reuses:

- `UIGenerator`
- `useFormEngine`
- preprocessing/postprocessing pipelines
- provider registries
- existing nonce propagation infrastructure

This minimizes long-term drift and keeps the playground behavior consistent with production runtime behavior.

---

The proposal is also grounded in direct repository investigation and local validation work, including:

- V2 runtime analysis
- CSP tracing
- provider registry investigation
- existing POC analysis
- CodeMirror feasibility validation
- local proof-of-concept experimentation

This significantly reduced architectural uncertainty before implementation planning.

---

Beyond the mentorship itself, the long-term goal is to help establish maintainable developer tooling infrastructure around the V2 plugin ecosystem.

The playground can evolve into:

- a primary onboarding surface for plugin authors
- a reusable validation environment
- a documentation platform
- a future schema-tooling foundation

for both OpenEverest maintainers and downstream contributors.

---

Ultimately, the goal is not simply to:

```txt
replace Monaco
```

but to build a secure, maintainable, extensible developer workflow platform that strengthens the long-term sustainability of the OpenEverest V2 plugin ecosystem.

# Appendix A — Existing PoC Validation Summary

---

# Purpose of the PoC

During proposal preparation, a CSP-safe proof-of-concept was implemented locally against the OpenEverest V2 branch.

The goal of this PoC was not to prematurely build the final playground, but to validate the highest-risk architectural assumptions early, especially around:

- CSP compatibility
- editor integration
- runtime reuse
- validation feasibility
- routing integration
- testing strategy

This substantially reduced implementation uncertainty before the mentorship period begins.

---

# Architectural Assumptions Validated

| Assumption | Result |
|---|---|
| CodeMirror 6 works under strict CSP | Confirmed |
| Existing nonce propagation flow is reusable | Confirmed |
| No backend CSP changes are required | Confirmed |
| `UIGenerator` can power live preview directly | Confirmed |
| Route-level integration is straightforward | Confirmed |
| Inline YAML diagnostics are feasible | Confirmed |
| Existing testing conventions adapt cleanly | Confirmed |

---

# PoC Capabilities

The prototype currently demonstrates:

- CodeMirror 6 integration
- nonce propagation
- YAML syntax highlighting
- split-pane layout
- route integration
- validation plumbing
- diagnostics rendering
- preview architecture
- test coverage

---

# CSP Validation Results

The PoC successfully demonstrated:

- zero CSP violations
- no `unsafe-eval`
- no blob workers
- no CSP policy relaxation
- nonce-safe style injection
- successful production-preview execution

This validated the core architectural direction behind the proposal.

---

# Testing Results

The PoC currently includes:

```txt
31 passing tests across 6 test files
```

covering:

- editor behavior
- nonce handling
- validation
- layout interactions
- accessibility behavior
- route rendering

This early testing effort helped validate repository integration assumptions before implementation planning.

---

# Why This Validation Matters

The proposal is therefore not based purely on theoretical architecture.

Several of the highest-risk assumptions have already been tested directly against the real OpenEverest frontend environment.

This provides:

- lower implementation uncertainty
- earlier architectural confidence
- clearer PR planning
- stronger maintainer reviewability

before the mentorship even begins.

# Appendix B — Key Design Decisions

---

# 1. CodeMirror 6 Instead of Monaco

The most important architectural decision in the proposal is replacing Monaco with CodeMirror 6.

The existing Monaco-based POC is incompatible with OpenEverest’s strict CSP because Monaco depends on:

- `new Function()`
- dynamic runtime evaluation
- worker-based execution

OpenEverest intentionally disallows these patterns through its production CSP.

---

## Why CodeMirror 6

CodeMirror 6 aligns naturally with the existing security model because it:

- avoids `unsafe-eval`
- avoids blob workers
- supports nonce propagation
- works cleanly under strict CSP
- integrates well with lightweight extension systems

---

## Trade-Off

CodeMirror provides fewer built-in IDE-style features than Monaco.

The proposal intentionally accepts this trade-off in favor of:

- security alignment
- maintainability
- production compatibility

---

# 2. Reusing the Production Runtime

The playground intentionally reuses the real runtime instead of introducing a parallel rendering system.

The preview directly integrates with:

- `UIGenerator`
- `useFormEngine`
- preprocess/postprocess pipelines
- provider registries

This ensures:

- runtime parity
- lower maintenance burden
- earlier bug discovery
- fewer architecture drift risks

---

# 3. Inline `mockData:` Design

Mock providers are embedded directly inside schemas:

```yaml
dataSource:
  provider: storage-classes
  mockData:
    - label: fast-storage
      value: fast-storage
```

This keeps playground exports:

- self-contained
- reproducible
- easy to share

---

## Trade-Off

This introduces playground-only fields into schemas.

Mitigation includes:

- export stripping
- warning diagnostics
- clear documentation

---

# 4. Conservative Validation Strategy

`TopologyUISchemas` intentionally includes flexible extension points.

The proposal therefore validates:

- top-level structure strictly
- nested component definitions loosely

This avoids rejecting valid future provider schemas while still surfacing meaningful diagnostics.

---

# 5. Additive Architecture

The implementation is intentionally additive.

### Added

- `/plugin-developer`
- CodeMirror integration
- validation layer
- persistence support
- mock-provider infrastructure

### Removed

- legacy Monaco-based POC

### Unchanged

- production routes
- CSP policy
- backend APIs
- runtime rendering flow

This minimizes regression risk during implementation.

---

# 6. Testing-First Development

The proposal emphasizes strong automated testing because the playground interacts with:

- CSP enforcement
- runtime rendering
- validation systems
- provider registries
- editor infrastructure

The implementation therefore includes:

- unit tests
- component tests
- Playwright E2E
- CSP-specific verification
- accessibility assertions

from the beginning rather than as final-stage polish.

---

# Final Design Philosophy

The overall architecture prioritizes:

- security
- runtime parity
- maintainability
- incremental delivery
- contributor friendliness

instead of introducing heavyweight abstractions or speculative complexity.

The goal is to build tooling that future contributors can understand and extend safely.

