from datetime import timedelta

from hatchet_sdk.utils.timedelta_to_expression import (
    timedelta_to_expr,
    expr_to_timedelta,
)


def test_timedelta_to_expr() -> None:
    assert timedelta_to_expr(timedelta(seconds=3600)) == "1h"
    assert timedelta_to_expr(timedelta(seconds=60)) == "1m"
    assert timedelta_to_expr(timedelta(seconds=1)) == "1s"
    assert timedelta_to_expr(timedelta(seconds=3661)) == "3661s"
    assert timedelta_to_expr(timedelta(hours=96)) == "96h"
    assert timedelta_to_expr(timedelta(hours=96, seconds=1)) == "345601s"


def test_expr_to_timedelta() -> None:
    assert expr_to_timedelta("1h") == timedelta(hours=1)
    assert expr_to_timedelta("1m") == timedelta(minutes=1)
    assert expr_to_timedelta("1s") == timedelta(seconds=1)
    assert expr_to_timedelta("3661s") == timedelta(seconds=3661)
    assert expr_to_timedelta("96h") == timedelta(hours=96)
    assert expr_to_timedelta("2d") == timedelta(days=2)
    assert expr_to_timedelta("2d") == timedelta(hours=48)
