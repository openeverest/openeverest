# Edge Cases & Pitfalls Explained

## Understanding the New Validation Logic

This document explains the corner cases and pitfalls that the strengthened validation handles, with examples and justifications.

---

## 1. The Major Version Boundary Problem

### The Bug
**Old code allowed**:
```
0.6.0 → 1.0.0
```

### Why It Happened
The old code only checked minor version:
```go
currentMinor := currentEverestVersion.Segments()[1]  // = 6
targetMinor := targetEverestVersion.Segments()[1]   // = 0
if targetMinor - currentMinor > 1 { ... }           // 0 - 6 = -6, not > 1 ✓ passes!
```

The check `targetMinor - currentMinor > 1` rejects only **forward jumps > 1 minor**, but:
- Doesn't care if major changed
- Doesn't check downgrade (that's a separate check)
- Allows any minor decrease if major increases

### Why It's Wrong
Major version bumps typically mean:
- **Breaking changes** in APIs/behavior
- **Different support lifecycle** (different version policies)
- **Different compatibility constraints** for dependencies
- **Manual migration** may be required

Example: `0.x` (beta) → `1.x` (stable) is a major milestone requiring human review.

### The Fix
```go
if targetMajor != currentMajor {
    return ErrMajorVersionChangeNotAllowed
}
```

Now `0.6.0 → 1.0.0` is **rejected** ❌

---

## 2. The Negative Minor Difference Silent Pass

### The Scenario
```
1.6.0 → 2.0.0
```

### What the Old Code Did
```go
currentMinor = 6
targetMinor = 0
if 0 - 6 > 1 {  // -6 > 1? NO ✓ passes!
    return error
}
```

The check passes because **-6 is not greater than 1**.

The check was designed to catch forward jumps like `0.6.0 → 0.8.0` (jump of +2), but fails to catch:
- Major change with minor downgrade: `1.6.0 → 2.0.0` (-6 minor, ignored)
- Major change with same minor: `1.6.0 → 2.6.0` (0 minor difference, ignored)
- Major change with forward minor: `1.6.0 → 2.8.0` (+2 minor, ignored)

### Why It's Wrong
The intention was to **prevent large version jumps in a single major version**. But by ignoring major changes, we silently allow the largest possible jump (major + any minor).

### The Fix
Check major version **first**:
```go
if targetMajor != currentMajor {
    return ErrMajorVersionChangeNotAllowed  // FAIL FAST
}
// Only then check minor within the same major
if targetMinor - currentMinor > 1 {
    return ErrCannotUpgradeByMoreThanOneMinorVersion
}
```

Now the major check catches the problem before minor check is evaluated. ✅

---

## 3. Index Out-of-Bounds on Malformed Versions

### The Scenario
```go
// What if someone tries a malformed version?
ver1 := goversion.Must(goversion.NewVersion("1"))     // Only 1 segment
ver2 := goversion.Must(goversion.NewVersion("1.5"))   // Only 2 segments (OK)

segments := ver1.Segments()  // []int{1}
minor := segments[1]         // ❌ PANIC: index out of range
```

### Why It Happened
The old code assumed `Segments()` always has at least 2 elements:
```go
currentMinor := currentEverestVersion.Segments()[1]  // Direct access
targetMinor := targetEverestVersion.Segments()[1]   // Direct access
```

### Why It's Wrong
While `goversion.NewVersion()` typically returns semantic versions (major.minor.patch), it's defensive to guard against:
- Versions with only major: `"1"`
- User input that somehow bypassed parsing validation
- Future API changes
- Edge cases in version service responses

### The Fix
```go
currentSegs := currentEverestVersion.Segments()
targetSegs := targetEverestVersion.Segments()

if len(currentSegs) < 2 || len(targetSegs) < 2 {
    return errors.New("invalid version format: must have major.minor")
}

currentMinor := currentSegs[1]  // Safe to access
targetMinor := targetSegs[1]    // Safe to access
```

Now `"1"` is **rejected with clear error** ✅

---

## 4. Prerelease Version Edge Case

### The Scenario
```
0.6.0 (stable) → 0.7.0-beta.1 (prerelease)
```

### How Go-Version Handles It
```go
stable := goversion.Must(goversion.NewVersion("0.6.0"))
beta := goversion.Must(goversion.NewVersion("0.7.0-beta.1"))

stable.LessThan(beta)  // true — stable < beta ✓
stable.Equal(beta)     // false
```

By semver spec, `0.7.0-beta.1 < 0.7.0` (prerelease is less than stable release).

### Validation Result
The code allows `0.6.0 → 0.7.0-beta.1`:
```
✓ Not a downgrade (0.6.0 < 0.7.0-beta.1)
✓ Not same version
✓ Major same (0 == 0)
✓ Minor jump +1 (7 - 6 = 1, not > 1)
```

**Result**: ✅ Allowed

### Is This Right?
Depends on project policy:

**✅ ALLOW prerelease** if:
- Team wants to test early
- User explicitly chose it
- Policy says users can opt-in to beta

**❌ REJECT prerelease** if:
- Production systems must avoid untested code
- Policy: only stable releases
- Want to guarantee "tested before release"

### How to Reject Prerelease
```go
if targetEverestVersion.Prerelease() != "" {
    return errors.New("cannot upgrade to prerelease version; use stable release")
}
```

Add test:
```go
{
    name: "reject prerelease target",
    currentEverestVersion: "0.6.0",
    targetEverestVersion: "0.7.0-beta.1",
    wantErrIs: errors.New("cannot upgrade to prerelease version..."),
}
```

The current implementation **allows prerelease** — if project policy changes, add the above check.

---

## 5. Build Metadata Edge Case

### The Scenario
```
0.6.0 → 0.6.0+build.123
0.6.0+build.456 → 0.6.0+build.789
```

### How Go-Version Handles It
Build metadata (`+...`) is **ignored** in version comparisons per semver:
```go
v1 := goversion.Must(goversion.NewVersion("0.6.0+build.123"))
v2 := goversion.Must(goversion.NewVersion("0.6.0"))

v1.Equal(v2)  // true — build metadata ignored ✓
```

### Validation Result
Both scenarios are **rejected** with `ErrNoUpdateAvailable`:
```
0.6.0 → 0.6.0+build.123:
✓ Not downgrade (0.6.0 == 0.6.0 ignoring metadata)
✓ Same version → ErrNoUpdateAvailable ✅ REJECTED
```

### Why This Is Correct
Build metadata is **not part of version semantics**. It's for build info (commit hash, build number, etc.). Two versions differing only in build metadata are semantically the same, so no upgrade is needed.

If you need to handle build metadata differently, you'd need to:
1. Parse and compare metadata manually (not recommended)
2. Policy decision: build metadata might indicate rollback to same version with different build

Current behavior is correct. ✅

---

## 6. Four-Segment and Five-Segment Versions

### The Scenario
```
0.6.0.1 → 0.7.0.0
1.2.3.4.5 → 1.3.0.0.0
```

### How Go-Version Handles It
```go
v := goversion.Must(goversion.NewVersion("0.6.0.1"))
segs := v.Segments()  // []int{0, 6, 0, 1}

// Code uses [0] and [1] only:
major := segs[0]  // 0
minor := segs[1]  // 6
// Ignores patch (2) and 4th segment (1)
```

### Validation Result
Extra segments are **ignored**:
```
0.6.0.1 → 0.7.0.0:
- Major: 0 → 0 ✓ same
- Minor: 6 → 7 ✓ +1 forward
→ ALLOWED ✅
```

### Is This Right?
Yes. The code explicitly checks only `segments[0]` and `segments[1]` (major.minor). 

Rationale:
- Patch and beyond are considered compatible within major.minor
- Go ecosystem often uses 4-segment versions (major.minor.patch.build)
- The upgrade policy is: "same major, forward at most 1 minor, any patch"

If future versions use semantic versioning strictly (3 segments), this works correctly.
If versions use extended format (4+ segments), this still works correctly.

---

## 7. Same-Version Upgrade with Different Patch

### The Scenario
```
0.6.0 → 0.6.1  (patch bump, same major.minor)
0.6.5 → 0.6.2  (patch downgrade within same minor)
```

### Validation Result
First case:
```
0.6.0 → 0.6.1:
✓ Not downgrade (0.6.0 < 0.6.1)
✓ Not same version (0.6.0 != 0.6.1)
✓ Major same (0 == 0)
✓ Minor same (6 == 6, not > 1)
→ ALLOWED ✅
```

Second case:
```
0.6.5 → 0.6.2:
✓ Is downgrade (0.6.5 > 0.6.2)
→ REJECTED with ErrDowngradeNotAllowed ✅
```

### Why This Is Right
- **Patch forward** (0.6.0 → 0.6.1): Bug fixes, security patches. Always allowed. ✅
- **Patch backward** (0.6.5 → 0.6.2): Breaking, could lose fixes. Rejected. ✅

This is correct behavior.

---

## 8. The Edge Case: Current at Higher Patch, Target at Lower Patch but Higher Minor

### The Scenario
```
0.6.5 → 0.7.0
```

### Validation Result
```
0.6.5 → 0.7.0:
✓ Not downgrade (0.6.5 < 0.7.0)  [semver: major.minor is primary]
✓ Not same version
✓ Major same (0 == 0)
✓ Minor forward +1 (7 - 6 = 1)
→ ALLOWED ✅
```

### Why This Is Right
In semver, **major.minor** determines version precedence. Patch is secondary.
```
0.6.5 < 0.7.0  (because 0.6 < 0.7)
```

Even though patch goes 5 → 0, the minor forward (6 → 7) dominates.

This is correct and safe. ✅

---

## Summary Table: All Edge Cases Handled

| Case | Example | Old Code | New Code | Correct? |
|------|---------|----------|----------|----------|
| **Major bump forward** | 0.6.0 → 1.0.0 | ✅ Allowed | ❌ Rejected | **NEW: Correct** |
| **Major bump with negative minor** | 1.6.0 → 2.0.0 | ✅ Allowed | ❌ Rejected | **NEW: Correct** |
| **Bounds check on 1-segment version** | "1" → "1.0" | ❌ Panic | ❌ Rejected cleanly | **NEW: Correct** |
| **Prerelease target** | 0.6.0 → 0.7.0-beta | ✅ Allowed | ✅ Allowed | Depends on policy |
| **Build metadata only** | 0.6.0 → 0.6.0+build | ✅ Rejected | ✅ Rejected | **Correct** |
| **4-segment version** | 0.6.0.1 → 0.7.0.2 | ✅ Allowed | ✅ Allowed | **Correct** |
| **Patch forward same minor** | 0.6.0 → 0.6.5 | ✅ Allowed | ✅ Allowed | **Correct** |
| **Patch backward same minor** | 0.6.5 → 0.6.2 | ❌ Allowed | ❌ Rejected | **NEW: Correct** |
| **Higher patch, lower minor forward** | 0.6.5 → 0.7.0 | ✅ Allowed | ✅ Allowed | **Correct** |

---

## Key Takeaway

The new validation is **strict but predictable**:
- **Rejects**: Major changes, multi-minor jumps, downgrades, same version, malformed versions
- **Allows**: Patch-only, forward one minor, forward one minor + patch
- **Safe**: Bounds checking, ordered validation, clear errors
- **Maintainable**: Defensive coding, explicit checks, documented rules
