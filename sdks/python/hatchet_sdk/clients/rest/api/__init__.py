# flake8: noqa

# import apis into api package

import importlib
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from hatchet_sdk.clients.rest.api.api_token_api import APITokenApi
    from hatchet_sdk.clients.rest.api.cel_api import CELApi
    from hatchet_sdk.clients.rest.api.default_api import DefaultApi
    from hatchet_sdk.clients.rest.api.durable_tasks_api import DurableTasksApi
    from hatchet_sdk.clients.rest.api.event_api import EventApi
    from hatchet_sdk.clients.rest.api.feature_flags_api import FeatureFlagsApi
    from hatchet_sdk.clients.rest.api.filter_api import FilterApi
    from hatchet_sdk.clients.rest.api.github_api import GithubApi
    from hatchet_sdk.clients.rest.api.healthcheck_api import HealthcheckApi
    from hatchet_sdk.clients.rest.api.log_api import LogApi
    from hatchet_sdk.clients.rest.api.metadata_api import MetadataApi
    from hatchet_sdk.clients.rest.api.observability_api import ObservabilityApi
    from hatchet_sdk.clients.rest.api.operator_api import OperatorApi
    from hatchet_sdk.clients.rest.api.rate_limits_api import RateLimitsApi
    from hatchet_sdk.clients.rest.api.sns_api import SNSApi
    from hatchet_sdk.clients.rest.api.slack_api import SlackApi
    from hatchet_sdk.clients.rest.api.step_run_api import StepRunApi
    from hatchet_sdk.clients.rest.api.task_api import TaskApi
    from hatchet_sdk.clients.rest.api.tenant_api import TenantApi
    from hatchet_sdk.clients.rest.api.user_api import UserApi
    from hatchet_sdk.clients.rest.api.webhook_api import WebhookApi
    from hatchet_sdk.clients.rest.api.worker_api import WorkerApi
    from hatchet_sdk.clients.rest.api.workflow_api import WorkflowApi
    from hatchet_sdk.clients.rest.api.workflow_run_api import WorkflowRunApi
    from hatchet_sdk.clients.rest.api.workflow_runs_api import WorkflowRunsApi

_LAZY_IMPORTS: dict[str, str] = {
    "APITokenApi": "hatchet_sdk.clients.rest.api.api_token_api",
    "CELApi": "hatchet_sdk.clients.rest.api.cel_api",
    "DefaultApi": "hatchet_sdk.clients.rest.api.default_api",
    "DurableTasksApi": "hatchet_sdk.clients.rest.api.durable_tasks_api",
    "EventApi": "hatchet_sdk.clients.rest.api.event_api",
    "FeatureFlagsApi": "hatchet_sdk.clients.rest.api.feature_flags_api",
    "FilterApi": "hatchet_sdk.clients.rest.api.filter_api",
    "GithubApi": "hatchet_sdk.clients.rest.api.github_api",
    "HealthcheckApi": "hatchet_sdk.clients.rest.api.healthcheck_api",
    "LogApi": "hatchet_sdk.clients.rest.api.log_api",
    "MetadataApi": "hatchet_sdk.clients.rest.api.metadata_api",
    "ObservabilityApi": "hatchet_sdk.clients.rest.api.observability_api",
    "OperatorApi": "hatchet_sdk.clients.rest.api.operator_api",
    "RateLimitsApi": "hatchet_sdk.clients.rest.api.rate_limits_api",
    "SNSApi": "hatchet_sdk.clients.rest.api.sns_api",
    "SlackApi": "hatchet_sdk.clients.rest.api.slack_api",
    "StepRunApi": "hatchet_sdk.clients.rest.api.step_run_api",
    "TaskApi": "hatchet_sdk.clients.rest.api.task_api",
    "TenantApi": "hatchet_sdk.clients.rest.api.tenant_api",
    "UserApi": "hatchet_sdk.clients.rest.api.user_api",
    "WebhookApi": "hatchet_sdk.clients.rest.api.webhook_api",
    "WorkerApi": "hatchet_sdk.clients.rest.api.worker_api",
    "WorkflowApi": "hatchet_sdk.clients.rest.api.workflow_api",
    "WorkflowRunApi": "hatchet_sdk.clients.rest.api.workflow_run_api",
    "WorkflowRunsApi": "hatchet_sdk.clients.rest.api.workflow_runs_api",
}

__all__ = list(_LAZY_IMPORTS)


def __getattr__(name: str) -> object:
    if name not in _LAZY_IMPORTS:
        raise AttributeError(f"module {__name__!r} has no attribute {name!r}")
    return getattr(importlib.import_module(_LAZY_IMPORTS[name]), name)


def __dir__() -> list[str]:
    return sorted(set(globals()) | set(_LAZY_IMPORTS))
