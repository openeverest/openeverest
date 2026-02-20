#!/usr/bin/env python3
"""
Copyright header fixer for the OpenEverest project.

Rules:
  1. File already has the OpenEverest copyright line → skip (idempotent).
  2. File has any Percona copyright line (any year / year-range variant) →
     insert the OpenEverest line immediately after that line.
  3. File has no copyright at all → prepend a full Apache 2.0 header with
     only the OpenEverest copyright.

Supported extensions: .go  .ts  .tsx

Exit codes:
  0 – no files were changed
  1 – one or more files were changed
"""

import os
import re
import sys
from datetime import date

YEAR = str(date.today().year)

OE_COPYRIGHT_LINE = f"// Copyright (C) {YEAR} The OpenEverest Contributors"

# Matches any Percona copyright variant, e.g.:
#   // Copyright (C) 2023 Percona LLC
#   // Copyright (C) 2023-2025 Percona LLC
#   // Copyright (C) 2023 Percona
PERCONA_RE = re.compile(r"^// Copyright \(C\) [\d]{4}(?:-[\d]{4})? Percona.*$", re.MULTILINE)

# Check whether the file already carries the OpenEverest line (any year).
OE_PRESENT_RE = re.compile(r"// Copyright \(C\) [\d]{4} The OpenEverest Contributors")

NEW_FILE_HEADER = f"""\
// Copyright (C) {YEAR} The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
"""

SUPPORTED_EXTENSIONS = {".go", ".ts", ".tsx"}


def fix_file(filepath: str) -> bool:
    """Return True if the file was modified."""
    try:
        with open(filepath, encoding="utf-8") as fh:
            content = fh.read()
    except (OSError, UnicodeDecodeError) as exc:
        print(f"WARNING: could not read {filepath}: {exc}", file=sys.stderr)
        return False

    # Rule 1 – already up-to-date.
    if OE_PRESENT_RE.search(content):
        return False

    percona_match = PERCONA_RE.search(content)

    if percona_match:
        # Rule 2 – insert OE line right after the matched Percona line.
        insert_pos = percona_match.end()
        new_content = content[:insert_pos] + "\n" + OE_COPYRIGHT_LINE + content[insert_pos:]
    else:
        # Rule 3 – no existing copyright; prepend the full header.
        # Keep one blank line between the new header and the existing content.
        new_content = NEW_FILE_HEADER + "\n" + content.lstrip("\n")

    with open(filepath, "w", encoding="utf-8") as fh:
        fh.write(new_content)

    print(f"Updated: {filepath}")
    return True


def iter_files(paths: list[str]):
    """Yield file paths, expanding directories recursively."""
    for path in paths:
        if os.path.isdir(path):
            for root, _, files in os.walk(path):
                for name in files:
                    yield os.path.join(root, name)
        elif os.path.isfile(path):
            yield path


def main() -> None:
    paths = sys.argv[1:]
    if not paths:
        print("Usage: add_copyright.py <file-or-dir> [...]")
        sys.exit(0)

    modified: list[str] = []
    for filepath in iter_files(paths):
        if os.path.splitext(filepath)[1] in SUPPORTED_EXTENSIONS:
            if fix_file(filepath):
                modified.append(filepath)

    if modified:
        print(f"\n{len(modified)} file(s) updated.")
        sys.exit(1)

    sys.exit(0)


if __name__ == "__main__":
    main()
