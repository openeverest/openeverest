# UI Monorepo: High-Level Overview

This document describes the main blocks of the UI monorepo,
their responsibilities, and dependency directions.

## Structural Diagram

```mermaid
flowchart LR
  subgraph APPS[Apps]
    A1[apps/everest]
  end

  subgraph PKGS[Packages]
    P1[packages/ui-lib]
    P2[packages/design]
    P3[packages/types]
    P4[packages/utils]
    P5[packages/plugin-sdk]
    P6[tooling packages\neslint-config, prettier-config, tsconfig]
  end

  subgraph TOOLING[Shared tooling]
    T1[pnpm workspace]
    T2[turborepo pipeline]
    T3[make workflows]
  end

  A1 --> P1
  A1 --> P2
  A1 --> P3
  A1 --> P4
  A1 --> P5
  A1 --> P6

  T1 --> A1
  T1 --> PKGS
  T2 --> A1
  T2 --> PKGS
  T3 --> T2
```

## Block Responsibilities

1. Apps:
   contain product applications; currently this is
   [../../../ui/apps/everest](../../../ui/apps/everest).
2. Packages:
   contain reusable UI libraries, design/tokens, types, utilities, and
   plugin contract SDKs.
3. Shared tooling:
   orchestrates builds, linting, testing, and shared workflows across workspaces.

## Boundaries and Dependency Direction

1. The application consumes libraries from packages.
2. Packages must not depend on the application.
3. Tooling orchestrates apps/packages and does not contain product logic.

## Metadata

- Owner: UI
- Last updated: 2026-06-11
