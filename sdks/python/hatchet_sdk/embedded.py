import atexit
import hashlib
import json
import os
import platform
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from hatchet_sdk.config import ClientConfig, ClientTLSConfig, EmbeddedHatchetConfig

REPO_URL = "https://github.com/hatchet-dev/hatchet-embedded"


@dataclass
class EmbeddedSidecar:
    token: str
    tenant_id: str
    grpc_address: str
    api_url: str
    process: subprocess.Popen[bytes]

    def stop(self) -> None:
        if self.process.poll() is not None:
            return

        self.process.terminate()
        try:
            self.process.wait(timeout=30)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()


def _sidecar_asset_name() -> str:
    system = {"darwin": "darwin", "linux": "linux"}.get(sys.platform)
    arch = {
        "arm64": "arm64",
        "aarch64": "arm64",
        "x86_64": "amd64",
        "amd64": "amd64",
    }.get(platform.machine().lower())

    if not system or not arch:
        raise RuntimeError(
            f"hatchet embedded is not supported on {sys.platform}/{platform.machine()}"
        )

    return f"hatchet-embedded-sidecar_{system}_{arch}"


class _NoRedirectHandler(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args: object, **kwargs: object) -> None:
        return None


def _resolve_version(version: str | None) -> str:
    requested = version or "latest"
    if requested != "latest":
        return requested

    opener = urllib.request.build_opener(_NoRedirectHandler)
    location = ""

    try:
        opener.open(f"{REPO_URL}/releases/latest")
    except urllib.error.HTTPError as e:
        location = e.headers.get("Location") or ""

    tag = location.rstrip("/").rsplit("/", 1)[-1]
    if not tag.startswith("v"):
        raise RuntimeError(
            f"could not resolve the latest hatchet-embedded release from {REPO_URL}"
        )

    return tag


def _expected_checksum(tag: str, asset: str) -> str:
    url = f"{REPO_URL}/releases/download/{tag}/checksums.txt"
    with urllib.request.urlopen(url, timeout=10) as res:
        checksums: str = res.read().decode()

    for line in checksums.splitlines():
        parts = line.split()
        if len(parts) == 2 and parts[1] == asset:
            return parts[0]

    raise RuntimeError(f"no checksum for {asset} in {url}")


def _resolve_expected_checksum(tag: str, asset: str, bin_path: Path) -> str:
    checksum_file = bin_path.with_name(asset + ".sha256")

    # fall back to the checksum cached at download time so a pinned, already
    # verified binary still starts when GitHub is unreachable
    try:
        expected = _expected_checksum(tag, asset)
    except (urllib.error.URLError, OSError):
        if bin_path.exists() and checksum_file.exists():
            return checksum_file.read_text().strip()
        raise

    checksum_file.write_text(expected + "\n")
    return expected


def _sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _ensure_sidecar_binary(version: str | None, checksum: str | None) -> Path:
    tag = _resolve_version(version)
    asset = _sidecar_asset_name()
    bin_path = Path.home() / ".hatchet" / "embedded" / tag / asset
    bin_path.parent.mkdir(parents=True, exist_ok=True)

    # verified on every start, not just at download; a cached binary that no
    # longer matches the expected checksum is re-downloaded
    expected = checksum or _resolve_expected_checksum(tag, asset, bin_path)

    if bin_path.exists() and _sha256_file(bin_path) == expected:
        return bin_path
    url = f"{REPO_URL}/releases/download/{tag}/{asset}"
    # unique temp file per call (not per process) so concurrent downloads of
    # the same version never clobber each other, even across threads; the
    # final rename is atomic and last-writer-wins
    tmp_fd, tmp_name = tempfile.mkstemp(
        prefix=f"{asset}.", suffix=".download", dir=bin_path.parent
    )
    os.close(tmp_fd)
    tmp_path = Path(tmp_name)

    try:
        urllib.request.urlretrieve(url, tmp_path)

        actual = _sha256_file(tmp_path)
        if actual != expected:
            raise RuntimeError(
                f"checksum mismatch for {url}: expected {expected}, got {actual}"
            )

        tmp_path.chmod(0o755)
        tmp_path.rename(bin_path)
    finally:
        tmp_path.unlink(missing_ok=True)

    return bin_path


def start_embedded_sidecar(options: EmbeddedHatchetConfig) -> EmbeddedSidecar:
    """
    Download (and cache) the hatchet-embedded sidecar binary, spawn it, and wait
    until the embedded engine is ready. The sidecar shuts down when this process
    exits. Use `Hatchet(config=ClientConfig(embedded=...))` unless you need the
    raw connection details.
    """
    if options.binary_path:
        if options.checksum:
            actual = _sha256_file(Path(options.binary_path))
            if actual != options.checksum:
                raise RuntimeError(
                    f"checksum mismatch for {options.binary_path}: expected {options.checksum}, got {actual}"
                )
        bin_path = options.binary_path
    else:
        bin_path = str(_ensure_sidecar_binary(options.version, options.checksum))
    handshake_path = (
        Path(tempfile.mkdtemp(prefix="hatchet-embedded-")) / "handshake.json"
    )

    args = [bin_path, "-handshake-file", str(handshake_path)]
    if options.database_url:
        args += ["-database-url", options.database_url]
    if options.rabbitmq_url:
        args += ["-rabbitmq-url", options.rabbitmq_url]
    if options.postgres_data_dir:
        args += ["-postgres-data-dir", options.postgres_data_dir]
    if options.grpc_port:
        args += ["-grpc-port", str(options.grpc_port)]
    if options.api_port:
        args += ["-api-port", str(options.api_port)]
    if not options.start_api:
        args += ["-no-api"]
    if not options.run_migrations:
        args += ["-no-migrations"]
    if options.log_level:
        args += ["-log-level", options.log_level]

    # the sidecar shuts down when its stdin closes, so it never outlives this process
    process = subprocess.Popen(args, stdin=subprocess.PIPE, stdout=subprocess.DEVNULL)
    atexit.register(process.terminate)

    try:
        handshake = _wait_for_handshake(
            process, handshake_path, options.ready_timeout_seconds
        )
    finally:
        handshake_path.unlink(missing_ok=True)
        handshake_path.parent.rmdir()

    return EmbeddedSidecar(
        token=handshake["token"],
        tenant_id=handshake["tenant_id"],
        grpc_address=handshake["grpc_address"],
        api_url=handshake["api_url"],
        process=process,
    )


def _wait_for_handshake(
    process: subprocess.Popen[bytes], handshake_path: Path, timeout_seconds: float
) -> dict[str, str]:
    deadline = time.monotonic() + timeout_seconds

    while time.monotonic() < deadline:
        if process.poll() is not None:
            raise RuntimeError(
                f"hatchet embedded sidecar exited with code {process.returncode} before becoming ready"
            )

        try:
            handshake = json.loads(handshake_path.read_text())
            if handshake.get("token"):
                return dict(handshake)
        except (OSError, json.JSONDecodeError):
            pass

        time.sleep(0.2)

    process.kill()
    raise TimeoutError(
        f"hatchet embedded sidecar did not become ready within {timeout_seconds}s"
    )


_HANDSHAKE_ENV = "HATCHET_EMBEDDED_HANDSHAKE"


def resolve_embedded_connection(config: ClientConfig) -> ClientConfig:
    """
    Start (or connect to) the embedded engine and return a `ClientConfig`
    pointing at it. `config.embedded` must be set.

    The result is built through the `ClientConfig` constructor (not
    `model_copy`), so its field/model validators run against the token the
    embedded engine just issued.
    """
    if config.embedded is None:
        raise ValueError(
            "config.embedded must be set to resolve an embedded connection"
        )

    # worker subprocesses re-import the main module; the handshake exported
    # below connects them to the parent's engine instead of booting a second one
    raw_handshake = os.environ.get(_HANDSHAKE_ENV)

    if raw_handshake is not None:
        handshake: dict[str, str] = json.loads(raw_handshake)
    else:
        sidecar = start_embedded_sidecar(config.embedded)
        handshake = {
            "token": sidecar.token,
            "tenant_id": sidecar.tenant_id,
            "grpc_address": sidecar.grpc_address,
            "api_url": sidecar.api_url,
        }
        os.environ[_HANDSHAKE_ENV] = json.dumps(handshake)

    connection: dict[str, Any] = {
        "token": handshake["token"],
        "tenant_id": handshake["tenant_id"],
        "host_port": handshake["grpc_address"],
        "tls_config": ClientTLSConfig(strategy="none"),
    }
    if handshake["api_url"]:
        connection["server_url"] = handshake["api_url"]

    return ClientConfig(**{**config.model_dump(), **connection})
