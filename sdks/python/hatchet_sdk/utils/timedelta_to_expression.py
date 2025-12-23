from datetime import timedelta

HOUR = 3600
MINUTE = 60

Duration = timedelta | str


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


def str_to_timedelta(s: str) -> timedelta:
    if s.endswith("h"):
        return timedelta(hours=int(s[:-1]))
    if s.endswith("m"):
        return timedelta(minutes=int(s[:-1]))
    if s.endswith("s"):
        return timedelta(seconds=int(s[:-1]))
    raise ValueError(f"Invalid duration string: {s}")
