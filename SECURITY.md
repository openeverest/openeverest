# Security Policy

Thank you for helping keep OpenEverest secure!

This policy covers every repository in the [openeverest](https://github.com/openeverest) organization: core, providers, plugins, Helm charts and tooling.

## Supported Versions

Security fixes are released for:

| Version | Supported |
| ------- | --------- |
| Latest v1.x minor release | :white_check_mark: |
| Latest v2 release (including pre-releases until v2 GA) | :white_check_mark: |
| Older releases | :x: |

Please upgrade to a supported version to receive fixes.

## Reporting a Vulnerability

**Do not report security issues in public GitHub issues, discussions, pull requests or Slack.**

Report privately through a [GitHub Security Advisory](https://github.com/openeverest/openeverest/security/advisories/new). If the issue is in a provider, plugin or other repository, you can open the advisory in that repository instead.

Please include:

- the affected component and version;
- a description of the vulnerability and its impact;
- steps to reproduce or a proof of concept, if possible;
- any known mitigations.

## Security Response Team

All [active maintainers](https://github.com/openeverest/governance/blob/main/MAINTAINERS.md) form the security response team. Reports and fixes are discussed privately, in the GitHub advisory and in closed maintainer sessions, as described in [GOVERNANCE.md](https://github.com/openeverest/governance/blob/main/GOVERNANCE.md#closed-sessions).

For each report, one maintainer acts as the **coordinator**. The coordinator:

- acknowledges the report and keeps the reporter informed;
- assesses severity together with the other maintainers;
- assigns the fix and drives the release;
- publishes the advisory.

## Response Process

| Step | Target |
| ---- | ------ |
| Acknowledge the report | 3 business days |
| Triage, confirm and assign severity ([CVSS v3.1](https://www.first.org/cvss/)) | 7 days |
| Fix released for **critical** and **high** severity | 30 days |
| Fix released for **medium** and **low** severity | next regular release |

1. **Triage.** The coordinator confirms the issue, determines affected versions and severity, and may invite the reporter to the private advisory.
2. **Fix.** The fix is developed in a private fork of the advisory, reviewed by at least one other maintainer, and backported to all supported versions.
3. **Disclosure.** We release patched versions, then publish the GitHub Security Advisory and request a CVE through GitHub. Releases are announced in the release notes and in [#openeverest-users](https://cloud-native.slack.com/archives/C09RRGZL2UX) on CNCF Slack.

## Embargo

We keep reports confidential until a fix is released. We aim to disclose within **90 days** of the report. If a fix needs more time, we agree on a new date with the reporter. If a vulnerability is already public or actively exploited, we may disclose earlier.

## Verifying Releases

Container images carry signed SLSA provenance and SBOM attestations. `everestctl` binaries ship with a cosign signature bundle (`<binary>.sigstore.json`) and SLSA provenance (`everestctl.intoto.jsonl`).

```sh
# Binary signature
cosign verify-blob everestctl-linux-amd64 \
  --bundle everestctl-linux-amd64.sigstore.json \
  --certificate-identity-regexp '^https://github.com/openeverest/openeverest/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# Binary or image provenance
gh attestation verify everestctl-linux-amd64 --repo openeverest/openeverest
gh attestation verify oci://ghcr.io/openeverest/openeverest:<version> --repo openeverest/openeverest
```

## Credit

We credit reporters in the advisory unless they ask to stay anonymous.

Thank you for contributing to a safer open source ecosystem!
