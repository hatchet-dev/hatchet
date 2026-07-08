from types import SimpleNamespace

from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.runnables.workflow import BaseWorkflow
from hatchet_sdk.types.trigger import TriggerWorkflowOptions


def _admin() -> AdminClient:
    # _prepare_workflow_request only uses the nested TriggerWorkflowRequest class
    # attribute, so we can bypass connection setup.
    return object.__new__(AdminClient)


def _admin_for_run() -> AdminClient:
    # _create_workflow_run_request additionally reads self.namespace and
    # self.config.apply_namespace; everything else comes from contextvars that
    # default to None. Stub just those two so we can drive the real request path.
    admin = object.__new__(AdminClient)
    admin.namespace = ""
    admin.config = SimpleNamespace(apply_namespace=lambda name, _ns: name)  # type: ignore[assignment]
    return admin


def test_prepare_workflow_request_maps_display_name() -> None:
    req = _admin()._prepare_workflow_request(
        "my-workflow", "{}", TriggerWorkflowOptions(display_name="Acme Corp")
    )

    assert req.HasField("display_name")
    assert req.display_name == "Acme Corp"


def test_prepare_workflow_request_omits_display_name_when_unset() -> None:
    req = _admin()._prepare_workflow_request(
        "my-workflow", "{}", TriggerWorkflowOptions()
    )

    assert not req.HasField("display_name")


def test_create_workflow_run_request_sends_display_name() -> None:
    # Regression guard: every public run surface (run/run_no_wait/aio_* and bulk)
    # and durable child spawns build their gRPC request via
    # _create_workflow_run_request, which reconstructs TriggerWorkflowOptions.
    # display_name must survive that reconstruction and reach the protobuf.
    req = _admin_for_run()._create_workflow_run_request(
        "my-workflow", "{}", TriggerWorkflowOptions(display_name="Acme Corp")
    )

    assert req.HasField("display_name")
    assert req.display_name == "Acme Corp"


def test_create_workflow_run_request_omits_display_name_when_unset() -> None:
    req = _admin_for_run()._create_workflow_run_request(
        "my-workflow", "{}", TriggerWorkflowOptions()
    )

    assert not req.HasField("display_name")


def test_direct_display_name_kwarg_flows_through_run_options() -> None:
    # The modern run()/run_many() API passes display_name as a direct keyword
    # argument (not via the deprecated `options` object). This is the shared funnel
    # every run surface routes through.
    workflow = object.__new__(BaseWorkflow)
    workflow._config = SimpleNamespace(default_additional_metadata={})  # type: ignore[assignment]

    options = workflow._create_trigger_run_options_with_combined_additional_meta(
        None, display_name="Acme Corp"
    )

    assert options.display_name == "Acme Corp"
