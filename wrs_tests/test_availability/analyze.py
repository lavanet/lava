#!/usr/bin/env python3
from __future__ import annotations

import subprocess
from pathlib import Path


def main() -> None:
    test_dir = Path(__file__).resolve().parent
    subprocess.run(
        [
            "python3",
            str(test_dir.parent / "_framework" / "analyze.py"),
            "--test-dir",
            str(test_dir),
        ],
        check=True,
    )


if __name__ == "__main__":
    main()

