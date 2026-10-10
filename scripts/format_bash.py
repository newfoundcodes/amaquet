#!/usr/bin/env python3
"""Safely check or format Bash scripts under tests/ with shfmt.

The default mode is read-only and exits with status 1 when formatting changes
would be made. Pass --write to apply the fully validated batch with atomic
per-file replacement and rollback on failure.
"""

from __future__ import annotations

import argparse
import difflib
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tempfile
from typing import NoReturn

REPOSITORY_ROOT = Path(__file__).resolve().parents[1]
DEFAULT_TARGET = REPOSITORY_ROOT / "tests"

class FormatError(RuntimeError):
    """Raised when formatting cannot be completed safely."""

def fail(message: str) -> NoReturn:
    raise FormatError(message)

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Check or format every Bash script under tests/ using shfmt.",
    )

    mode = parser.add_mutually_exclusive_group()
    mode.add_argument(
        "--check",
        action="store_true",
        help="report formatting differences without writing (the default)",
    )
    mode.add_argument(
        "--write",
        action="store_true",
        help="apply formatting with atomic replacement after all scripts pass validation",
    )

    parser.add_argument(
        "--target",
        type=Path,
        default=DEFAULT_TARGET,
        help="directory to scan (default: repository tests/ directory)",
    )
    parser.add_argument(
        "--shfmt",
        default=os.environ.get("SHFMT", "shfmt"),
        help="shfmt executable or path (default: SHFMT or shfmt)",
    )
    parser.add_argument(
        "--bash",
        default=os.environ.get("BASH", "bash"),
        help="Bash executable or path (default: BASH or bash)",
    )
    parser.add_argument(
        "--indent",
        type=int,
        default=4,
        help="indent width passed to shfmt (default: 4)",
    )
    parser.add_argument(
        "--no-diff",
        action="store_true",
        help="list files that need formatting without printing unified diffs",
    )

    return parser.parse_args()

def resolve_executable(value: str, label: str) -> str:
    candidate = Path(value).expanduser()
    if candidate.parent != Path(".") or candidate.is_absolute():
        if not candidate.is_file() or not os.access(candidate, os.X_OK):
            fail(f"{label} executable is not usable: {candidate}")
        return str(candidate.resolve())

    resolved = shutil.which(value)
    if resolved is None:
        fail(
            f"{label} executable {value!r} was not found in PATH. "
            "Install mvdan.cc/sh/v3/cmd/shfmt or pass --shfmt /path/to/shfmt."
        )
    return resolved

def discover_scripts(target: Path) -> list[Path]:
    target = target.expanduser().resolve()
    try:
        target.relative_to(REPOSITORY_ROOT)
    except ValueError:
        fail(f"refusing to format outside the repository: {target}")

    if not target.is_dir():
        fail(f"target directory does not exist: {target}")

    scripts = sorted(path for path in target.rglob("*.sh") if path.is_file())
    if not scripts:
        fail(f"no .sh files found under {target}")

    for path in scripts:
        if path.is_symlink():
            fail(f"refusing to format symbolic link: {path}")
    return scripts

def run_checked(command: list[str], *, input_bytes: bytes | None = None) -> subprocess.CompletedProcess[bytes]:
    try:
        return subprocess.run(
            command,
            cwd=REPOSITORY_ROOT,
            input=input_bytes,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
    except OSError as exc:
        fail(f"could not run {command[0]!r}: {exc}")

def validate_script(bash: str, path: Path) -> None:
    result = run_checked([bash, "-n", str(path)])
    if result.returncode != 0:
        diagnostic = result.stderr.decode("utf-8", errors="replace").strip()
        fail(f"Bash syntax check failed for {path.relative_to(REPOSITORY_ROOT)}:\n{diagnostic}")

def validate_bytes(bash: str, source_path: Path, content: bytes) -> None:
    with tempfile.NamedTemporaryFile(
        prefix="amaquet-format-bash-",
        suffix=".sh",
        dir=source_path.parent,
        delete=False,
    ) as handle:
        temporary = Path(handle.name)
        handle.write(content)
        handle.flush()
        os.fsync(handle.fileno())

    try:
        validate_script(bash, temporary)
    except FormatError as exc:
        fail(f"formatted output for {source_path.relative_to(REPOSITORY_ROOT)} is invalid:\n{exc}")
    finally:
        temporary.unlink(missing_ok=True)

def format_script(shfmt: str, path: Path, content: bytes, indent: int) -> bytes:
    result = run_checked(
        [
            shfmt,
            "-filename",
            str(path),
            "-ln",
            "bash",
            "-i",
            str(indent),
            "-ci",
        ],
        input_bytes=content,
    )

    if result.returncode != 0:
        diagnostic = result.stderr.decode("utf-8", errors="replace").strip()
        fail(f"shfmt failed for {path.relative_to(REPOSITORY_ROOT)}:\n{diagnostic}")
    return result.stdout

def atomic_write(path: Path, content: bytes, mode: int) -> None:
    descriptor, temporary_name = tempfile.mkstemp(
        prefix=f".{path.name}.",
        suffix=".tmp",
        dir=path.parent,
    )
    temporary = Path(temporary_name)

    try:
        with os.fdopen(descriptor, "wb") as handle:
            handle.write(content)
            handle.flush()
            os.fsync(handle.fileno())
        os.chmod(temporary, mode)
        os.replace(temporary, path)
    finally:
        temporary.unlink(missing_ok=True)

def display_diff(path: Path, original: bytes, formatted: bytes) -> None:
    relative = path.relative_to(REPOSITORY_ROOT).as_posix()
    try:
        before = original.decode("utf-8").splitlines(keepends=True)
        after = formatted.decode("utf-8").splitlines(keepends=True)
    except UnicodeDecodeError as exc:
        fail(f"{relative} is not valid UTF-8: {exc}")

    sys.stdout.writelines(
        difflib.unified_diff(
            before,
            after,
            fromfile=f"a/{relative}",
            tofile=f"b/{relative}",
        )
    )

def apply_batch(changes: dict[Path, bytes], originals: dict[Path, bytes], modes: dict[Path, int]) -> None:
    applied: list[Path] = []
    try:
        for path, content in changes.items():
            if path.read_bytes() != originals[path]:
                fail(
                    f"refusing to overwrite {path.relative_to(REPOSITORY_ROOT)}: "
                    "the file changed while formatting was in progress"
                )
            atomic_write(path, content, modes[path])
            applied.append(path)
    except BaseException as exc:
        rollback_errors: list[str] = []
        for path in reversed(applied):
            try:
                if path.read_bytes() != changes[path]:
                    rollback_errors.append(
                        f"{path}: changed externally after formatting; left untouched"
                    )
                    continue
                atomic_write(path, originals[path], modes[path])
            except OSError as rollback_exc:
                rollback_errors.append(f"{path}: {rollback_exc}")

        if rollback_errors:
            fail(
                f"formatting failed ({exc}); rollback also failed:\n"
                + "\n".join(rollback_errors)
            )

        if isinstance(exc, (KeyboardInterrupt, SystemExit)):
            raise

        fail(f"formatting failed; all modified files were restored: {exc}")

def main() -> int:
    args = parse_args()
    if args.indent < 0 or args.indent > 16:
        fail("--indent must be between 0 and 16")

    shfmt = resolve_executable(args.shfmt, "shfmt")
    bash = resolve_executable(args.bash, "Bash")
    scripts = discover_scripts(args.target)
    version = run_checked([shfmt, "--version"])

    if version.returncode != 0:
        fail(version.stderr.decode("utf-8", errors="replace").strip() or "shfmt --version failed")

    originals: dict[Path, bytes] = {}
    modes: dict[Path, int] = {}

    for path in scripts:
        validate_script(bash, path)

        originals[path] = path.read_bytes()
        modes[path] = stat.S_IMODE(path.stat().st_mode)

    changes: dict[Path, bytes] = {}
    for path in scripts:
        formatted = format_script(shfmt, path, originals[path], args.indent)
        validate_bytes(bash, path, formatted)

        if formatted != originals[path]:
            changes[path] = formatted

    if not changes:
        print(f"Bash formatting is clean: {len(scripts)} scripts checked")
        return 0

    if not args.write:
        if not args.no_diff:
            for path, formatted in changes.items():
                display_diff(path, originals[path], formatted)

        print(
            f"Bash formatting required: {len(changes)} of {len(scripts)} scripts",
            file=sys.stderr,
        )

        print("Run scripts/format_bash.py --write to apply the validated batch.", file=sys.stderr)
        return 1

    apply_batch(changes, originals, modes)
    print(f"Formatted {len(changes)} of {len(scripts)} Bash scripts")
    return 0

if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except FormatError as exc:
        print(f"format_bash.py: error: {exc}", file=sys.stderr)
        raise SystemExit(2)
