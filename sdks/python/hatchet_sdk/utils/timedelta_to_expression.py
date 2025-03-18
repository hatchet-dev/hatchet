from datetime import timedelta


def timedelta_to_expr(td: timedelta) -> str:
    return f"{td.seconds}s"
