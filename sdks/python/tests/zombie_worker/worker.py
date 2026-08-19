from datetime import timedelta
import ctypes

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()


@hatchet.task(execution_timeout=timedelta(seconds=5))
def die(input: EmptyModel, ctx: Context) -> None:
    ctypes.string_at(0)
