from datetime import timedelta

from hatchet_sdk.utils.timedelta_to_expression import timedelta_to_expr


def test_timedelta_to_expr():
    assert timedelta_to_expr(timedelta(seconds=3600)) == "1h"
    assert timedelta_to_expr(timedelta(seconds=60)) == "1m"
    assert timedelta_to_expr(timedelta(seconds=1)) == "1s"
    assert timedelta_to_expr(timedelta(seconds=3661)) == "3661s"
    assert timedelta_to_expr(timedelta(hours=96)) == "96h"
    assert timedelta_to_expr(timedelta(hours=96, seconds=1)) == "345601s"
