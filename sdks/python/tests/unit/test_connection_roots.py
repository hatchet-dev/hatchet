import base64
import hashlib
import pkgutil
import re

import pytest

from hatchet_sdk.connection import (
    _DISTRUSTED_ROOT_SHA256,
    _filter_distrusted_roots,
    _uses_bundled_roots,
)

_PEM = re.compile(
    rb"-----BEGIN CERTIFICATE-----(.+?)-----END CERTIFICATE-----", re.DOTALL
)
_GTS_ROOT_R4_SHA256 = "71cca5391f9e794b04802530b363e121da8a3043bb26662fea4dca7fc951a4bd"


def test_filter_drops_distrusted_roots() -> None:
    bundle = pkgutil.get_data("grpc", "_cython/_credentials/roots.pem")
    assert bundle

    filtered, _ = _filter_distrusted_roots(bundle)

    found = {
        hashlib.sha256(base64.b64decode(b"".join(m.split()))).hexdigest()
        for m in _PEM.findall(filtered)
    }

    assert not found & _DISTRUSTED_ROOT_SHA256
    assert _GTS_ROOT_R4_SHA256 in found


@pytest.mark.parametrize(
    ("path", "system_roots", "expected"),
    [
        (None, None, True),
        (None, "", True),
        (None, "0", True),
        (None, "on", True),
        (None, "1", False),
        (None, "t", False),
        (None, "TRUE", False),
        (None, " y ", False),
        (None, "Yes", False),
        ("/etc/roots.pem", None, False),
    ],
)
def test_uses_bundled_roots_env_gating(
    monkeypatch: pytest.MonkeyPatch,
    path: str | None,
    system_roots: str | None,
    expected: bool,
) -> None:
    monkeypatch.delenv("GRPC_DEFAULT_SSL_ROOTS_FILE_PATH", raising=False)
    monkeypatch.delenv("GRPC_USE_SYSTEM_ROOTS_OVER_LANGUAGE_CALLBACK", raising=False)

    if path is not None:
        monkeypatch.setenv("GRPC_DEFAULT_SSL_ROOTS_FILE_PATH", path)

    if system_roots is not None:
        monkeypatch.setenv("GRPC_USE_SYSTEM_ROOTS_OVER_LANGUAGE_CALLBACK", system_roots)

    assert _uses_bundled_roots() is expected
