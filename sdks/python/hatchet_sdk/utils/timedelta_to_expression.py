import warnings
from datetime import timedelta

HOUR = 3600
MINUTE = 60

Duration = timedelta | str


def _warn_if_str_duration(*durations: Duration | None, stacklevel: int = 3) -> None:
    if any(isinstance(d, str) for d in durations):
        warnings.warn(
            "Support for providing a duration as a string is deprecated and will be removed in v2.0.0. Use `timedelta` instead.",
            DeprecationWarning,
            stacklevel=stacklevel,
        )


def timedelta_to_expr(td: Duration) -> str:
    if isinstance(td, str):
        return td

    ## `total_seconds` gives the entire duration,
    ## while `seconds` gives the seconds component of the timedelta
    ## e.g. 1 day and 1 second would give 86401 total seconds but 1 second
    seconds = int(td.total_seconds())

    ## IMPORTANT: We only support hours, minutes, and seconds on the engine
    if seconds % HOUR == 0:
        return f"{seconds // HOUR}h"
    if seconds % MINUTE == 0:
        return f"{seconds // MINUTE}m"
    return f"{seconds}s"
