from datetime import timedelta


def timedelta_to_expr(td: timedelta | str) -> str:
    if isinstance(td, str):
        return td

    return f"{td.seconds}s"
