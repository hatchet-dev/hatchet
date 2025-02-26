import traceback


def errorWithTraceback(message: str, e: Exception):
    trace = "".join(traceback.format_exception(type(e), e, e.__traceback__))
    return f"{message}\n{trace}"
