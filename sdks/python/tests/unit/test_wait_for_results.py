from datetime import timedelta

from hatchet_sdk.conditions import (
    Condition,
    OrGroup,
    SleepCondition,
    UserEventCondition,
    flatten_conditions,
    or_,
)
from hatchet_sdk.context.context import (
    EventWaitResult,
    OrGroupResult,
    SleepResult,
    _build_condition_meta,
    _parse_wait_for_payload,
)


def test_each_or_group_yields_one_or_group_result_even_with_a_single_condition() -> (
    None
):
    conditions: list[Condition | OrGroup] = [
        or_(UserEventCondition(event_key="approved")),
        or_(
            SleepCondition(duration=timedelta(seconds=30)),
            UserEventCondition(event_key="rejected"),
        ),
        UserEventCondition(event_key="audit"),
    ]
    meta = _build_condition_meta(conditions, flatten_conditions(conditions))

    results = _parse_wait_for_payload(
        {
            "CREATE": {
                "approved": [{"by": "alice"}],
                "sleep:30s": [{"sleep_duration": "30s"}],
                "audit": [{"id": 1}],
            }
        },
        meta,
    )

    assert results == [
        OrGroupResult(result=EventWaitResult(key="approved", payload={"by": "alice"})),
        OrGroupResult(result=SleepResult(duration=timedelta(seconds=30))),
        EventWaitResult(key="audit", payload={"id": 1}),
    ]
