from datetime import timedelta

DAY = 86400
HOUR = 3600
MINUTE = 60

Duration = timedelta | str


def timedelta_to_expr(td: Duration) -> str:
    if isinstance(td, str):
        return td

    seconds = td.seconds

    if seconds % DAY == 0:
        return f"{seconds // DAY}d"
    elif seconds % HOUR == 0:
        return f"{seconds // HOUR}h"
    elif seconds % MINUTE == 0:
        return f"{seconds // MINUTE}m"
    else:
        return f"{seconds}s"
