import base64
import hashlib
import re

from hatchet_sdk.connection import DISTRUSTED_ROOT_SHA256, _filtered_bundled_roots

_PEM = re.compile(
    rb"-----BEGIN CERTIFICATE-----(.+?)-----END CERTIFICATE-----", re.DOTALL
)
_GTS_ROOT_R4_SHA256 = "71cca5391f9e794b04802530b363e121da8a3043bb26662fea4dca7fc951a4bd"


def test_bundled_roots_exclude_distrusted_roots() -> None:
    filtered = _filtered_bundled_roots()
    assert filtered

    found = {
        hashlib.sha256(base64.b64decode(b"".join(m.split()))).hexdigest()
        for m in _PEM.findall(filtered)
    }

    assert not found & DISTRUSTED_ROOT_SHA256
    assert _GTS_ROOT_R4_SHA256 in found
