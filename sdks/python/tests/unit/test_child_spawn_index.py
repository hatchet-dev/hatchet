from unittest.mock import MagicMock

from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.runnables.contextvars import ctx_action_key, workflow_spawn_indices
from hatchet_sdk.types.trigger import TriggerWorkflowOptions


def test_children_spawned_from_a_task_are_indexed_from_zero() -> None:
    config = MagicMock()
    config.apply_namespace.side_effect = lambda name, _namespace: name
    admin_client = AdminClient(config, MagicMock(), MagicMock())

    action_key = "test-action-key"
    token = ctx_action_key.set(action_key)
    try:
        child_indices = [
            admin_client._create_workflow_run_request(
                "child", None, TriggerWorkflowOptions()
            ).child_index
            for _ in range(3)
        ]
    finally:
        ctx_action_key.reset(token)
        workflow_spawn_indices.pop(action_key, None)

    assert child_indices == [0, 1, 2]
