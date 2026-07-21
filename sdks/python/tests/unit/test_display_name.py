"""Unit tests for the definition-level ``display_name`` CEL expression.

``display_name`` is declared on the workflow/task *definition* and holds a CEL
expression that the engine evaluates against the run's input at trigger time. It
mirrors the ``concurrency`` option and flows into two proto fields:

- workflow-level ``display_name`` -> ``CreateWorkflowVersionRequest.display_name``
- task-level ``display_name`` -> ``CreateTaskOpts.display_name``

Both fields are ``optional`` in the proto, so absence is asserted via
``HasField("display_name")``.
"""

from hatchet_sdk import Hatchet
from hatchet_sdk.config import ClientConfig


def _hatchet() -> Hatchet:
    # model_construct skips the JWT validation that a real token would require;
    # defining workflows and calling to_proto() is a pure, offline operation.
    return Hatchet(config=ClientConfig.model_construct(namespace="", token=""))


def test_workflow_display_name_maps_to_proto() -> None:
    h = _hatchet()
    workflow = h.workflow(name="customer", display_name="input.customerName")

    req = workflow.to_proto()

    assert req.HasField("display_name")
    assert req.display_name == "input.customerName"


def test_task_display_name_maps_to_proto() -> None:
    h = _hatchet()
    workflow = h.workflow(name="customer")

    @workflow.task(display_name="'enrich-' + input.name")
    def enrich(input, ctx):  # type: ignore[no-untyped-def]
        return {}

    req = workflow.to_proto()

    assert len(req.tasks) == 1
    assert req.tasks[0].HasField("display_name")
    assert req.tasks[0].display_name == "'enrich-' + input.name"


def test_display_name_unset_is_absent() -> None:
    h = _hatchet()
    workflow = h.workflow(name="customer")

    @workflow.task()
    def do_thing(input, ctx):  # type: ignore[no-untyped-def]
        return {}

    req = workflow.to_proto()

    assert not req.HasField("display_name")
    assert not req.tasks[0].HasField("display_name")


def test_standalone_task_display_name_is_task_level() -> None:
    # A single-task run declared via ``hatchet.task`` carries the display_name at
    # the *task* level; the engine's precedence uses it to name the run.
    h = _hatchet()

    @h.task(name="single", display_name="input.x")
    def single(input, ctx):  # type: ignore[no-untyped-def]
        return {}

    req = single._workflow.to_proto()

    assert not req.HasField("display_name")
    assert req.tasks[0].HasField("display_name")
    assert req.tasks[0].display_name == "input.x"


def test_standalone_durable_task_display_name_is_task_level() -> None:
    h = _hatchet()

    @h.durable_task(name="single_durable", display_name="input.y")
    async def single_durable(input, ctx):  # type: ignore[no-untyped-def]
        return {}

    req = single_durable._workflow.to_proto()

    assert not req.HasField("display_name")
    assert req.tasks[0].HasField("display_name")
    assert req.tasks[0].display_name == "input.y"
