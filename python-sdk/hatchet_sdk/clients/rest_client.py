from typing import Any, Dict

from .rest.api.log_api import LogApi
from .rest.api.step_run_api import StepRunApi
from .rest.api.workflow_api import WorkflowApi
from .rest.api.workflow_run_api import WorkflowRunApi
from .rest.api_client import ApiClient
from .rest.configuration import Configuration
from .rest.models import TriggerWorkflowRunRequest


class RestApi:
    def __init__(self, host: str, api_key: str, tenant_id: str):
        self.tenant_id = tenant_id

        config = Configuration(
            host=host,
            access_token=api_key,
        )

        # Create an instance of the API client
        api_client = ApiClient(configuration=config)
        self.workflow_api = WorkflowApi(api_client)
        self.workflow_run_api = WorkflowRunApi(api_client)
        self.step_run_api = StepRunApi(api_client)
        self.log_api = LogApi(api_client)

    def workflow_list(self):
        return self.workflow_api.workflow_list(
            tenant=self.tenant_id,
        )

    def workflow_get(self, workflow_id: str):
        return self.workflow_api.workflow_get(
            workflow=workflow_id,
        )

    def workflow_version_get(self, workflow_id: str, version: str | None = None):
        return self.workflow_api.workflow_version_get(
            workflow=workflow_id,
            version=version,
        )

    def workflow_run_list(
        self,
        workflow_id: str | None = None,
        offset: int | None = None,
        limit: int | None = None,
        event_id: str | None = None,
    ):
        return self.workflow_api.workflow_run_list(
            tenant=self.tenant_id,
            offset=offset,
            limit=limit,
            workflow_id=workflow_id,
            event_id=event_id,
        )

    def workflow_run_get(self, workflow_run_id: str):
        return self.workflow_api.workflow_run_get(
            tenant=self.tenant_id,
            workflow_run=workflow_run_id,
        )

    def workflow_run_create(self, workflow_id: str, input: Dict[str, Any]):
        return self.workflow_run_api.workflow_run_create(
            workflow=workflow_id,
            trigger_workflow_run_request=TriggerWorkflowRunRequest(
                input=input,
            ),
        )

    def list_logs(self, step_run_id: str):
        return self.log_api.log_line_list(
            step_run=step_run_id,
        )
