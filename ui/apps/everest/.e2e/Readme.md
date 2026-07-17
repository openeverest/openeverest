# E2E tests

## How to Run Locally

**in `openeverest` dir**

- `make k3d-cluster-up` - it will create K3D cluster and place kubeconfig into ~/.kube/ dir

- `make deploy-all` - will build everest packages (CLI, server), prepare docker images (server, operator) and deploy all this into your current K8S cluster (current context in kubeconfig -> K3D cluster you created on prev step)

**in `everest/ui/apps/everest/.e2e`**

- `make ci-init` - it will install tests dependencies and install an additional stuff into everest deployment (additional DB namespaces, MinIO, PMM helm chart, ...)

- `make test` - run UI e2e tests

---

## Visual Baseline Tests (`pr:visual` project)

Visual baseline tests capture screenshots of key UI states and compare them pixel-by-pixel on subsequent runs. This helps detect unintended visual regressions (e.g. after MUI version upgrades, theme changes, or layout refactoring).

### Prerequisites

- Dev environment running (`make dev` in `ui/` or Tilt)
- UI available at `http://localhost:3000`
- At least one DB instance exists in the cluster (tests reference `inst-u3y` in `default` namespace)

### Compile tests

From `ui/apps/everest`:

```bash
pnpm pre-e2e
```

### Generate / update baseline screenshots

Generates (or overwrites) all `.png` baseline files in `pr/visual/visual-baseline.e2e.ts-snapshots/`:

```bash
cd .e2e
EVEREST_URL=http://localhost:3000 CI_USER=admin CI_PASSWORD=admin \
  npx playwright test --project=pr:visual --update-snapshots --workers=1
```

### Run in comparison mode (verify screenshots match)

Compares current UI against existing baselines. Fails if any screenshot differs beyond the allowed threshold (`maxDiffPixelRatio: 0.01`):

```bash
cd .e2e
EVEREST_URL=http://localhost:3000 CI_USER=admin CI_PASSWORD=admin \
  npx playwright test --project=pr:visual --workers=1
```

### Run a single test

```bash
npx playwright test --project=pr:visual -g "Databases list page" --workers=1
```

### Key flags

| Flag                 | Purpose                                                |
| -------------------- | ------------------------------------------------------ |
| `--update-snapshots` | Overwrite baseline `.png` files with current state     |
| _(no flag)_          | Compare against existing baselines — fails on mismatch |
| `--reporter=line`    | Compact single-line output per test                    |
| `-g "pattern"`       | Run only tests matching the name pattern               |

### What happens on failure

When comparison mode detects a mismatch, Playwright writes three files into `test-results/`:

- `*-expected.png` — the baseline
- `*-actual.png` — what was rendered
- `*-diff.png` — highlighted pixel differences

### Notes

- Tests run sequentially (`fullyParallel: false`, `workers: 1`) to avoid token expiry issues
- Some tests mock API responses via `page.route()` to guarantee deterministic UI state
- The `pr:visual` project is **not** a dependency of `pr` — it won't run on CI unless explicitly requested
- Baselines are platform-specific (suffix includes `-pr-visual-linux.png`)
