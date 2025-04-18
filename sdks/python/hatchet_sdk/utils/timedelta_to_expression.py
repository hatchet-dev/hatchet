from datetime import timedelta

HOUR = 3600
MINUTE = 60

Duration = timedelta | str


def timedelta_to_expr(td: Duration) -> str:
    ## IMPORTANT: We only support hours, minutes, and seconds on the engine
    if isinstance(td, str):
        return td

    seconds = td.seconds

    if seconds % HOUR == 0:
        return f"{seconds // HOUR}h"
    elif seconds % MINUTE == 0:
        return f"{seconds // MINUTE}m"
    else:
        return f"{seconds}s"
