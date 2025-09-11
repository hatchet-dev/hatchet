from enum import Enum


class OTelAttribute(str, Enum):
    ## Shared
    NAMESPACE = "namespace"
    ADDITIONAL_METADATA = "additional_metadata"
    WORKFLOW_NAME = "workflow_name"

    PRIORITY = "priority"

    ## Unfortunately named - this corresponds to all types of payloads, not just actions
    ACTION_PAYLOAD = "payload"

    ## Action
    ACTION_NAME = "action_name"
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
    WORKFLOW_RUN_ID = "workflow_run_id"
    WORKFLOW_VERSION_ID = "workflow_version_id"

    ## Push Event
    EVENT_KEY = "event_key"
    FILTER_SCOPE = "scope"

    ## Trigger Workflow
    PARENT_ID = "parent_id"
    PARENT_STEP_RUN_ID = "parent_step_run_id"
    CHILD_INDEX = "child_index"
    CHILD_KEY = "child_key"
    DESIRED_WORKER_ID = "desired_worker_id"
    STICKY = "sticky"
    KEY = "key"

    ## Schedule Workflow
    RUN_AT_TIMESTAMPS = "run_at_timestamps"
