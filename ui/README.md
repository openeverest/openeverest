# OpenEverest Frontend

This code base is a monorepo that holds a number of packages, namely:

- `@percona/utils` -> common shared utils code (e.g. `kebabize` string function);
- `@percona/types` -> common shared types;
- `@percona/design` -> Percona UI themes and related utils, from base to Everest;
- `@percona/ui-lib` -> common UI library;
- `@percona/eslint-config-react`, `@percona/prettier-config`, `@percona/tsconfig` -> packages for shared configurations of ESlint, Prettier and Typescript;
- `@percona/everest` -> The OpenEverest frontend code.

OpenEverest is an open source Database-as-a-Service solution that automates day-one and day-two operations for Postgres, MySQL, and MongoDB databases within Kubernetes clusters.

## Pre-Requisites

Make sure PNPM is installed: https://pnpm.io/installation

## Stack

This repo uses the following stack across its packages:

- PNPM to manage workspaces in the monorepo and dependencies (https://pnpm.io/);
- Typescript (https://www.typescriptlang.org/);
- React (https://react.dev/);
- Rollup to bundle the different common packages (https://rollupjs.org/);
- Vite for Everest development (https://vitejs.dev/);
- Vitest for unit tests (https://vitest.dev/);
- Playwright for E2E tests (https://playwright.dev/);
- Turborepo for orchestration across packages and caching (https://turbo.build/);

## Install dependencies

```bash
make init
```

## Run everest in development mode

```bash
make dev
```

## Using backup locations credentials locally for E2E tests

Please rename the `.env.test` to `.env` in `apps/everest/.e2e` folder and add the correct values for the backup location.

## Build packages for production

```bash
make build
```

By default, Everest build will be placed under `apps/everest/dist`. This can be changed using `EVEREST_OUT_DIR`:

```bash
make build EVEREST_OUT_DIR=my-dist
```

This path can either be relative to the repo root or absolute.

## Run a command for a particular workspace

```bash
pnpm --filter <workspace> <command>
```

More about PNPM filtering: https://pnpm.io/filtering

## Run CI E2E tests locally

- Make sure to have the following setup on `<repo-root>/dev/config.yaml`:

  ```
  namespaces:
    - name: pxc-only
      backupStorages:
        - bucket-3
      operators:
        - pxc

    - name: psmdb-only
      backupStorages:
        - bucket-2
      operators:
        - psmdb

    - name: pg-only
      backupStorages:
        - bucket-4
      operators:
        - pg

    - name: everest-ui
      backupStorages:
        - bucket-1
        - bucket-5
      operators:
        - pxc
        - psmdb
        - pg

  ```

- On `apps/everest/.e2e/.env`, set:

  ```
  EVEREST_BUCKETS_NAMESPACES_MAP='[["bucket-1","everest-ui"],["bucket-2","psmdb-only"],["bucket-3","pxc-only"],["bucket-4","pg-only"],["bucket-5","everest-ui"]]'
  ```

- To set a MinIO storage on your k8s cluster: `kubectl apply -f <repo-root>/.github/minio.conf.yaml` and set:

  - `EVEREST_LOCATION_ACCESS_KEY=minioadmin`
  - `EVEREST_LOCATION_SECRET_KEY=minioadmin`
  - `EVEREST_LOCATION_REGION=us-east-1`
  - `EVEREST_LOCATION_URL=https://minio.minio.svc.cluster.local`

- To setup a monitoring instance (PMM) on your k8s cluster:

  ```
  helm install pmm --set secret.pmm_password='admin',service.type=ClusterIP percona/pmm

  url=$(kubectl get svc/monitoring-service -o json | jq -r '.spec.clusterIP')

  echo -n "MONITORING_URL=http://$url"
  ```

  - Use the IP from the output and set:
    - `MONITORING_URL=<OUTPUT_IP>`
    - `MONITORING_USER=admin`
    - `MONITORING_PASSWORD=admin`

- Finally, run the tests using one of:
  - `pnpm --filter "@percona/everest" e2e`, to run all tests, including RBAC
  - ` pnpm --filter "@percona/everest" e2e:ignore-rbac`, to skip RBAC tests
