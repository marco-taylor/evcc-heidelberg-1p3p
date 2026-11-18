#!/usr/bin/env python3
"""Run hard regressions from an immutable, SHA256-verified firmware candidate.

This is host fault injection without device access, not hardware approval.
"""

import argparse
import hashlib
from pathlib import Path
import subprocess
import sys
import tempfile
import urllib.request

REPOSITORY = "marco-taylor/heidelberg-1p3p-ethernet"
COMMIT = "4682a33e7852a995c9112705ae63503125735b18"
FILES = {
    "src/switch.cpp": "d6d00c82083a4cf5ff9252e4b61e03e312ce1cc21a7c49cf5815774d14efd1aa",
    "src/pages.cpp": "978bccecc7d6d8248bd8864de0c33c569deaeba2bc6724d8bbf8de6ae2a6e761",
    "include/switch.h": "0a6ee123284d8a6853cdec216c3a7f2c854ee91f7cf5d3ec773110ea1ebaaf19",
    "test/host/safety_sim.py": "dc2db6104b024020c734e11079c3449a792c52b568603b75cabad15e51d78141",
}


def run(root):
    for filename, expected in FILES.items():
        actual = hashlib.sha256((root / filename).read_bytes()).hexdigest()
        if actual != expected:
            raise ValueError(f"SHA256 mismatch: {filename}: {actual}")
    print(f"Verified firmware {REPOSITORY}@{COMMIT}", flush=True)
    return subprocess.run([sys.executable, str(root / "test/host/safety_sim.py"),
                           "--firmware-root", str(root)], check=False).returncode


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--firmware-root", type=Path)
    mode.add_argument("--fetch-pinned", action="store_true")
    args = parser.parse_args()
    if args.firmware_root:
        return run(args.firmware_root.resolve())
    with tempfile.TemporaryDirectory(prefix="heidelberg-verified-") as directory:
        root = Path(directory)
        for filename in FILES:
            url = f"https://raw.githubusercontent.com/{REPOSITORY}/{COMMIT}/{filename}"
            with urllib.request.urlopen(url, timeout=30) as response:
                data = response.read(2_000_001)
            if len(data) > 2_000_000:
                raise ValueError(f"Unexpected download size: {filename}")
            path = root / filename
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        return run(root)


if __name__ == "__main__":
    raise SystemExit(main())
