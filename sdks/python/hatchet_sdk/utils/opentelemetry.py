from enum import Enum


class OTelAttribute(str, Enum):
    ## Action
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

    ## Push Event
    EVENT_KEY = "push_event.key"
    EVENT_PAYLOAD = "push_event.payload"
    EVENT_ADDITIONAL_METADATA = "push_event.additional_metadata"
    EVENT_NAMESPACE = "push_event.namespace"
    EVENT_PRIORITY = "push_event.priority"
    EVENT_SCOPE = "push_event.scope"

    ## Trigger Workflow
    RUN_WORKFLOW_WORKFLOW_NAME = "run_workflow.workflow_name"
    RUN_WORKFLOW_PAYLOAD = "run_workflow.payload"
    RUN_WORKFLOW_PARENT_ID = "run_workflow.parent_id"
    RUN_WORKFLOW_PARENT_STEP_RUN_ID = "run_workflow.parent_step_run_id"
    RUN_WORKFLOW_CHILD_INDEX = "run_workflow.child_index"
    RUN_WORKFLOW_CHILD_KEY = "run_workflow.child_key"
    RUN_WORKFLOW_NAMESPACE = "run_workflow.namespace"
    RUN_WORKFLOW_ADDITIONAL_METADATA = "run_workflow.additional_metadata"
    RUN_WORKFLOW_PRIORITY = "run_workflow.priority"
    RUN_WORKFLOW_DESIRED_WORKER_ID = "run_workflow.desired_worker_id"
    RUN_WORKFLOW_STICKY = "run_workflow.sticky"
    RUN_WORKFLOW_KEY = "run_workflow.key"
