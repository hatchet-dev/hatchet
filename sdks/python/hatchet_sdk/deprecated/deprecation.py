"""Generic time-aware deprecation helper.

Timeline (from a given start date, with configurable windows):
  0 to warn_days:              WARNING logged once per feature
  warn_days to error_days:     ERROR logged once per feature
  after error_days:            raises DeprecationError 1-in-5 calls (20% chance)

Defaults: warn_days=90, error_days=None (error phase disabled unless explicitly set).
"""

import random
from datetime import datetime, timezone

from hatchet_sdk.logger import logger

_DEFAULT_WARN_DAYS = 90

# Tracks which features have already been logged (keyed by feature name).
_already_logged: set[str] = set()


class DeprecationError(Exception):
    """Raised when a deprecation grace period has expired."""


def parse_semver(v: str) -> tuple[int, int, int]:
    """Parse a semver string like ``"v0.78.23"`` into ``(major, minor, patch)``.

    Returns ``(0, 0, 0)`` if parsing fails.
    """
    v = v.lstrip("v").split("-", 1)[0]
    parts = v.split(".")
    if len(parts) != 3:
        return (0, 0, 0)
    try:
        return (int(parts[0]), int(parts[1]), int(parts[2]))
    except ValueError:
        return (0, 0, 0)


def semver_less_than(a: str, b: str) -> bool:
    """Return ``True`` if semver string *a* is strictly less than *b*."""
    return parse_semver(a) < parse_semver(b)


def emit_deprecation_notice(
    feature: str,
    message: str,
    start: datetime,
    *,
    warn_days: int = _DEFAULT_WARN_DAYS,
    error_days: int | None = None,
) -> None:
    """Emit a time-aware deprecation notice.

    Args:
        feature:    A short identifier for the deprecated feature (used for
                    deduplication so each feature only logs once per process).
        message:    The human-readable deprecation message.
        start:      The UTC datetime when the deprecation window began.
        warn_days:  Days after *start* during which a warning is logged (default 90).
        error_days: Days after *start* during which an error is logged.
                    After this window, calls have a 20% chance of raising.
                    If None (default), the error/raise phase is never reached —
                    the notice stays at error-level logging indefinitely.

    Raises:
        DeprecationError: After the error_days window, raised ~20% of the time.
    """
    now = datetime.now(tz=timezone.utc)
    days_since = (now - start).days

    if days_since < warn_days:
        # Phase 1: warning
        if feature not in _already_logged:
            logger.warning(message)
            _already_logged.add(feature)

    elif error_days is None or days_since < error_days:
        # Phase 2: error-level log (indefinite when error_days is None)
        if feature not in _already_logged:
            logger.error(
                f"{message} " "This fallback will be removed soon. Upgrade immediately."
            )
            _already_logged.add(feature)

    else:
        # Phase 3: raise 1-in-5 times
        if feature not in _already_logged:
            logger.error(
                f"{message} "
                "This fallback is no longer supported and will fail intermittently."
            )
            _already_logged.add(feature)

        if random.random() < 0.2:
            raise DeprecationError(f"{feature}: {message}")
