import atexit
import hashlib
import json
import multiprocessing
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
from typing import TYPE_CHECKING

from pydantic import BaseModel

if TYPE_CHECKING:
    from hatchet_sdk.hatchet import Hatchet

REPO_URL = "https://github.com/hatchet-dev/hatchet-embedded"
DEFAULT_READY_TIMEOUT_SECONDS = 300.0


class EmbeddedOptions(BaseModel):
    version: str | None = None
    """
    hatchet-embedded release tag to download (defaults to HATCHET_EMBEDDED_VERSION or
    latest). Tags correspond to the Hatchet engine version baked into the sidecar,
    so pinning this pins the engine.
    """

    binary_path: str | None = None
    """path to an existing sidecar binary, skips the download (or HATCHET_EMBEDDED_BINARY_PATH)"""

    checksum: str | None = None
    """
    expected sha256 hex digest of the sidecar binary. When set, it replaces the
    release's checksums.txt as the trust anchor, so a compromised release
    channel cannot substitute the binary. Pin it together with `version`.
    """

    database_url: str | None = None
    """use an existing Postgres instead of the bundled one"""

    postgres_data_dir: str | None = None
    """store the bundled Postgres runtime and data under this directory"""

    grpc_port: int | None = None
    api_port: int | None = None

    start_api: bool = True
    """set to False to start only the engine + gRPC, no REST API"""

    run_migrations: bool = True
    """set to False to skip running migrations on startup"""

    rabbitmq_url: str | None = None
    """use RabbitMQ instead of the Postgres message queue"""

    log_level: str | None = None
    ready_timeout_seconds: float = DEFAULT_READY_TIMEOUT_SECONDS


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
    requested = version or os.environ.get("HATCHET_EMBEDDED_VERSION") or "latest"
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
    with urllib.request.urlopen(url) as res:
        checksums: str = res.read().decode()

    for line in checksums.splitlines():
        parts = line.split()
        if len(parts) == 2 and parts[1] == asset:
            return parts[0]

    raise RuntimeError(f"no checksum for {asset} in {url}")


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

    # verified on every start, not just at download; a cached binary that no
    # longer matches the expected checksum is re-downloaded
    expected = checksum or _expected_checksum(tag, asset)

    if bin_path.exists() and _sha256_file(bin_path) == expected:
        return bin_path

    bin_path.parent.mkdir(parents=True, exist_ok=True)
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


def start_embedded_sidecar(options: EmbeddedOptions | None = None) -> EmbeddedSidecar:
    """
    Download (and cache) the hatchet-embedded sidecar binary, spawn it, and wait
    until the embedded engine is ready. The sidecar shuts down when this process
    exits. Use `HatchetEmbedded()` unless you need the raw connection details.
    """
    options = options or EmbeddedOptions()

    bin_path = (
        options.binary_path
        or os.environ.get("HATCHET_EMBEDDED_BINARY_PATH")
        or str(_ensure_sidecar_binary(options.version, options.checksum))
    )
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


def HatchetEmbedded(options: EmbeddedOptions | None = None) -> "Hatchet":  # noqa: N802
    """
    Run a full Hatchet engine locally via the hatchet-embedded sidecar
    (downloaded on first use) and return a client wired to it. By default
    the sidecar starts a bundled Postgres; pass `database_url` in the
    options to point it at your own instead.

    :param options: Options for the embedded engine (version, ports, database, ...).
    :return: A Hatchet client instance connected to the embedded engine.
    """
    from hatchet_sdk.config import ClientConfig, ClientTLSConfig
    from hatchet_sdk.hatchet import Hatchet

    # worker subprocesses re-import the main module; connect them to the
    # parent's engine (via the env vars exported below) instead of booting
    # a second one. parent_process() is None while a spawn child is still
    # importing the main module, so also check the _inheriting flag set
    # during that phase.
    in_child = multiprocessing.parent_process() is not None or getattr(
        multiprocessing.current_process(), "_inheriting", False
    )
    if in_child:
        return Hatchet()

    sidecar = start_embedded_sidecar(options)

    os.environ["HATCHET_CLIENT_TOKEN"] = sidecar.token
    os.environ["HATCHET_CLIENT_TENANT_ID"] = sidecar.tenant_id
    os.environ["HATCHET_CLIENT_HOST_PORT"] = sidecar.grpc_address
    os.environ["HATCHET_CLIENT_TLS_STRATEGY"] = "none"
    if sidecar.api_url:
        os.environ["HATCHET_CLIENT_SERVER_URL"] = sidecar.api_url

    server_url = {"server_url": sidecar.api_url} if sidecar.api_url else {}

    return Hatchet(
        config=ClientConfig(
            token=sidecar.token,
            tenant_id=sidecar.tenant_id,
            host_port=sidecar.grpc_address,
            tls_config=ClientTLSConfig(strategy="none"),
            **server_url,  # type: ignore[arg-type]
        )
    )
