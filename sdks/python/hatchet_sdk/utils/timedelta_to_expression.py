from datetime import timedelta

HOUR = 3600
MINUTE = 60


def timedelta_to_expr(td: timedelta) -> str:
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


def expr_to_timedelta(expr: str) -> timedelta:
    unit = expr[-1]
    value = int(expr[:-1])

    if unit == "d":
        return timedelta(days=value)
    if unit == "h":
        return timedelta(hours=value)
    if unit == "m":
        return timedelta(minutes=value)
    if unit == "s":
        return timedelta(seconds=value)

    raise ValueError(f"Invalid time expression: {expr}")
