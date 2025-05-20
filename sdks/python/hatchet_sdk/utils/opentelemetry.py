from enum import Enum


class OTelAttribute(str, Enum):
    ACTION_NAME = "action_name"
    ACTION_PAYLOAD = "action_payload"
    CHILD_WORKFLOW_INDEX = "child_workflow_index"
    CHILD_WORKFLOW_KEY = "child_workflow_key"
    GET_GROUP_KEY_RUN_ID = "get_group_key_run_id"
    PARENT_WORKFLOW_RUN_ID = "parent_workflow_run_id"
    RETRY_COUNT = "retry_count"
    STEP_ID = "step_id"
    STEP_RUN_ID = "step_run_id"
    TENANT_ID = "tenant_id"
    WORKER_ID = "worker_id"
    WORKFLOW_ID = "workflow_id"
    WORKFLOW_NAME = "workflow_name"
    WORKFLOW_RUN_ID = "workflow_run_id"
    WORKFLOW_VERSION_ID = "workflow_version_id"
