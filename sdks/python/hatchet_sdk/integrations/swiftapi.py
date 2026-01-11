"""
SwiftAPI integration for Hatchet SDK.

Provides execution governance for AI agent task execution. When enabled,
every task execution is verified by SwiftAPI before running, creating
cryptographic attestations that prove authorization.

Usage:
    from hatchet_sdk import Hatchet
    from hatchet_sdk.integrations.swiftapi import SwiftAPIConfig

    hatchet = Hatchet()

    # Enable SwiftAPI for a worker
    worker = hatchet.worker(
        "my-worker",
        slots=10,
        swiftapi=SwiftAPIConfig(
            api_key="swiftapi_live_...",
        ),
    )
"""

from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Any

from hatchet_sdk.logger import logger

try:
    from swiftapi import Enforcement, PolicyViolation, SwiftAPI

    SWIFTAPI_AVAILABLE = True
except ImportError:
    SWIFTAPI_AVAILABLE = False
    SwiftAPI = None  # type: ignore[assignment, misc]
    Enforcement = None  # type: ignore[assignment, misc]
    PolicyViolation = None  # type: ignore[assignment, misc]


@dataclass
class SwiftAPIConfig:
    """Configuration for SwiftAPI integration.

    :param api_key: SwiftAPI API key (starts with swiftapi_live_ or swiftapi_test_)
    :param paranoid: If True, always verify attestation revocation online
    :param verbose: If True, print verification status messages
    :param action_prefix: Prefix for action types (default: "hatchet")
    :param on_violation: Callback when policy violation occurs
    """

    api_key: str
    paranoid: bool = False
    verbose: bool = False
    action_prefix: str = "hatchet"
    on_violation: Callable[[str, str, Exception], None] | None = None
    _enforcement: Any = field(default=None, init=False, repr=False)

    def __post_init__(self) -> None:
        if not SWIFTAPI_AVAILABLE:
            raise ImportError(
                "swiftapi-python is not installed. Install it with: pip install swiftapi-python"
            )

        if not self.api_key:
            raise ValueError("SwiftAPI api_key is required")

        if not self.api_key.startswith("swiftapi_"):
            raise ValueError(
                "Invalid SwiftAPI key format. Keys should start with 'swiftapi_live_' or 'swiftapi_test_'"
            )

        client = SwiftAPI(key=self.api_key)
        self._enforcement = Enforcement(
            client=client,
            paranoid=self.paranoid,
            verbose=self.verbose,
        )

    def verify_task_execution(
        self,
        action_name: str,
        workflow_name: str,
        workflow_run_id: str,
        step_run_id: str,
        worker_id: str,
        tenant_id: str,
        retry_count: int,
    ) -> dict[str, Any]:
        """Verify task execution with SwiftAPI.

        :param action_name: The action being executed
        :param workflow_name: Name of the workflow
        :param workflow_run_id: Workflow run ID
        :param step_run_id: Step run ID
        :param worker_id: Worker ID executing the task
        :param tenant_id: Tenant ID
        :param retry_count: Current retry count

        :returns: Attestation data if approved

        :raises PolicyViolation: If the action is denied by policy
        """
        action_type = f"{self.action_prefix}.task.execute"
        intent = f"Execute task '{action_name}' in workflow '{workflow_name}'"

        params = {
            "action_name": action_name,
            "workflow_name": workflow_name,
            "workflow_run_id": workflow_run_id,
            "step_run_id": step_run_id,
            "worker_id": worker_id,
            "tenant_id": tenant_id,
            "retry_count": retry_count,
        }

        try:
            return self._enforcement.client.verify(
                action_type=action_type,
                intent=intent,
                params=params,
            )
        except PolicyViolation as e:
            logger.warning(
                "SwiftAPI denied task execution: action=%s, workflow=%s, reason=%s",
                action_name,
                workflow_name,
                getattr(e, "denial_reason", str(e)),
            )
            if self.on_violation:
                self.on_violation(action_name, workflow_name, e)
            raise


def is_swiftapi_available() -> bool:
    """Check if SwiftAPI SDK is installed."""
    return SWIFTAPI_AVAILABLE
