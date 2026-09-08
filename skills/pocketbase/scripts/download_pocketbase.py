#!/usr/bin/env python3
"""Download and extract the official PocketBase binary for the current platform."""

import argparse
import hashlib
import json
import os
import platform
import re
import shutil
import stat
import sys
import tempfile
import urllib.error
import urllib.request
import zipfile
from pathlib import Path


GITHUB_API_BASE = "https://api.github.com/repos/pocketbase/pocketbase"
DEFAULT_OUTPUT_DIR = Path(".cache/pocketbase")
USER_AGENT = "pocketbase-skill-downloader"


class DownloadError(Exception):
    """Raised when the PocketBase download workflow cannot complete."""


def parse_args():
    parser = argparse.ArgumentParser(
        description=(
            "Download the latest PocketBase binary for the current platform, "
            "verify its checksum, and extract it locally."
        )
    )
    parser.add_argument(
        "--version",
        default="latest",
        help="PocketBase version to download, for example 0.37.1. Defaults to latest.",
    )
    parser.add_argument(
        "--output-dir",
        default=str(DEFAULT_OUTPUT_DIR),
        help="Directory where the extracted binary should be stored.",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Redownload and reextract even if the target binary already exists.",
    )
    return parser.parse_args()


def request_bytes(url):
    request = urllib.request.Request(
        url,
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": USER_AGENT,
        },
    )
    try:
        with urllib.request.urlopen(request) as response:
            return response.read()
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8", errors="replace")
        raise DownloadError(
            "Request failed for {url}: HTTP {status} {reason}\n{body}".format(
                url=url,
                status=error.code,
                reason=error.reason,
                body=body.strip(),
            )
        ) from error
    except urllib.error.URLError as error:
        raise DownloadError("Request failed for {url}: {reason}".format(url=url, reason=error.reason)) from error


def request_json(url):
    return json.loads(request_bytes(url).decode("utf-8"))


def normalize_version(version):
    if version == "latest":
        return version
    return version.lstrip("v")


def resolve_platform_asset_parts():
    system_name = platform.system().lower()
    machine_name = platform.machine().lower()

    supported = {
        "darwin": {
            "x86_64": "amd64",
            "amd64": "amd64",
            "arm64": "arm64",
            "aarch64": "arm64",
        },
        "linux": {
            "x86_64": "amd64",
            "amd64": "amd64",
            "arm64": "arm64",
            "aarch64": "arm64",
            "armv7l": "armv7",
            "armv7": "armv7",
            "ppc64le": "ppc64le",
            "s390x": "s390x",
        },
        "windows": {
            "x86_64": "amd64",
            "amd64": "amd64",
            "arm64": "arm64",
            "aarch64": "arm64",
        },
    }

    if system_name not in supported or machine_name not in supported[system_name]:
        raise DownloadError(
            "Unsupported PocketBase platform: system={system} machine={machine}".format(
                system=system_name,
                machine=machine_name,
            )
        )

    return system_name, supported[system_name][machine_name]


def fetch_release(version):
    if version == "latest":
        return request_json(GITHUB_API_BASE + "/releases/latest")

    tag_name = version if version.startswith("v") else "v" + version
    return request_json(GITHUB_API_BASE + "/releases/tags/" + tag_name)


def release_version(release):
    return release["tag_name"].lstrip("v")


def expected_asset_name(version, system_name, arch_name):
    return "pocketbase_{version}_{system}_{arch}.zip".format(
        version=version,
        system=system_name,
        arch=arch_name,
    )


def select_asset(release, asset_name):
    for asset in release.get("assets", []):
        if asset.get("name") == asset_name:
            return asset
    raise DownloadError("Release does not contain asset {name}".format(name=asset_name))


def select_checksums_asset(release):
    for asset in release.get("assets", []):
        if asset.get("name") == "checksums.txt":
            return asset
    raise DownloadError("Release does not contain checksums.txt")


def parse_checksums(checksums_text):
    checksums = {}
    pattern = re.compile(r"^(?:sha256:)?([0-9a-fA-F]{64})\s+[* ]?(.+)$")

    for raw_line in checksums_text.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        match = pattern.match(line)
        if match:
            checksum, filename = match.groups()
            checksums[filename.strip()] = checksum.lower()

    return checksums


def sha256_of_file(path):
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def ensure_executable(path):
    if os.name == "nt":
        return
    current_mode = path.stat().st_mode
    path.chmod(current_mode | stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH)


def extract_binary(archive_path, destination_path):
    binary_name = "pocketbase.exe" if os.name == "nt" else "pocketbase"

    with zipfile.ZipFile(archive_path) as archive:
        selected_member = None
        for member in archive.infolist():
            if member.is_dir():
                continue
            if Path(member.filename).name == binary_name:
                selected_member = member
                break

        if selected_member is None:
            raise DownloadError(
                "Archive {archive} does not contain {binary}".format(
                    archive=archive_path,
                    binary=binary_name,
                )
            )

        destination_path.parent.mkdir(parents=True, exist_ok=True)
        with archive.open(selected_member) as source, destination_path.open("wb") as target:
            shutil.copyfileobj(source, target)

    ensure_executable(destination_path)


def main():
    args = parse_args()
    requested_version = normalize_version(args.version)
    output_root = Path(args.output_dir).expanduser().resolve()

    system_name, arch_name = resolve_platform_asset_parts()
    release = fetch_release(requested_version)
    version = release_version(release)
    asset_name = expected_asset_name(version, system_name, arch_name)
    platform_dir = "{system}_{arch}".format(system=system_name, arch=arch_name)
    binary_name = "pocketbase.exe" if system_name == "windows" else "pocketbase"
    binary_path = output_root / version / platform_dir / binary_name

    if binary_path.exists() and not args.force:
        print("Resolved release: v{version}".format(version=version))
        print("Selected asset: {asset}".format(asset=asset_name))
        print("Binary path: {path}".format(path=binary_path))
        print(str(binary_path))
        return 0

    asset = select_asset(release, asset_name)
    checksums_asset = select_checksums_asset(release)

    with tempfile.TemporaryDirectory(prefix="pocketbase-download-") as temp_dir:
        temp_path = Path(temp_dir)
        archive_path = temp_path / asset_name

        archive_path.write_bytes(request_bytes(asset["browser_download_url"]))
        checksums_text = request_bytes(checksums_asset["browser_download_url"]).decode(
            "utf-8",
            errors="replace",
        )

        checksums = parse_checksums(checksums_text)
        expected_checksum = checksums.get(asset_name)
        if expected_checksum is None:
            raise DownloadError(
                "checksums.txt does not contain an entry for {asset}".format(asset=asset_name)
            )

        actual_checksum = sha256_of_file(archive_path)
        if actual_checksum != expected_checksum:
            raise DownloadError(
                "Checksum mismatch for {asset}: expected {expected}, got {actual}".format(
                    asset=asset_name,
                    expected=expected_checksum,
                    actual=actual_checksum,
                )
            )

        if binary_path.exists():
            binary_path.unlink()

        extract_binary(archive_path, binary_path)

    print("Resolved release: v{version}".format(version=version))
    print("Selected asset: {asset}".format(asset=asset_name))
    print("Binary path: {path}".format(path=binary_path))
    print(str(binary_path))
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except DownloadError as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
