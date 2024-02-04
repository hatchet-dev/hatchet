# relative imports
from ..dispatcher_pb2 import GroupKeyActionEvent, StepActionEvent, ActionEventResponse, ActionType, AssignedAction, WorkerListenRequest, WorkerRegisterRequest, WorkerUnsubscribeRequest, WorkerRegisterResponse
from ..dispatcher_pb2_grpc import DispatcherStub

import time
from ..loader import ClientConfig
from ..logger import logger
import json
import grpc
from typing import Callable, List, Union
from ..metadata import get_metadata


def new_dispatcher(conn, config: ClientConfig):
    return DispatcherClientImpl(
        client=DispatcherStub(conn),
        token=config.token,
    )
    
class DispatcherClient:
    def get_action_listener(self, ctx, req):
        raise NotImplementedError

    def send_step_action_event(self, ctx, in_):
        raise NotImplementedError

DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 1  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5
DEFAULT_ACTION_TIMEOUT = 60  # seconds
DEFAULT_REGISTER_TIMEOUT = 5

class GetActionListenerRequest:
    def __init__(self, worker_name: str, services: List[str], actions: List[str]):
        self.worker_name = worker_name
        self.services = services
        self.actions = actions

class Action:
    def __init__(self, worker_id: str, tenant_id: str, workflow_run_id: str, get_group_key_run_id: str, job_id: str, job_name: str, job_run_id: str, step_id: str, step_run_id: str, action_id: str, action_payload: str, action_type: ActionType):
        self.worker_id = worker_id
        self.workflow_run_id = workflow_run_id
        self.get_group_key_run_id = get_group_key_run_id
        self.tenant_id = tenant_id
        self.job_id = job_id
        self.job_name = job_name
        self.job_run_id = job_run_id
        self.step_id = step_id
        self.step_run_id = step_run_id
        self.action_id = action_id
        self.action_payload = action_payload
        self.action_type = action_type

class WorkerActionListener:
    def actions(self, ctx, err_ch):
        raise NotImplementedError

    def unregister(self):
        raise NotImplementedError

START_STEP_RUN = 0
CANCEL_STEP_RUN = 1
START_GET_GROUP_KEY = 2

class ActionListenerImpl(WorkerActionListener):
    def __init__(self, client : DispatcherStub, token, worker_id):
        self.client = client
        self.token = token
        self.worker_id = worker_id
        self.retries = 0
        
        # self.logger = logger
        # self.validator = validator

    def actions(self):
        while True:
            logger.info("Listening for actions...")

            try:
                for assigned_action in self.get_listen_client():
                    assigned_action : AssignedAction

                    # Process the received action
                    action_type = self.map_action_type(assigned_action.actionType)

                    if assigned_action.actionPayload is None or assigned_action.actionPayload == "":
                        action_payload = None
                    else:
                        action_payload = self.parse_action_payload(assigned_action.actionPayload)

                    action = Action(
                        tenant_id=assigned_action.tenantId,
                        worker_id=self.worker_id,
                        workflow_run_id=assigned_action.workflowRunId,
                        get_group_key_run_id=assigned_action.getGroupKeyRunId,
                        job_id=assigned_action.jobId,
                        job_name=assigned_action.jobName,
                        job_run_id=assigned_action.jobRunId,
                        step_id=assigned_action.stepId,
                        step_run_id=assigned_action.stepRunId,
                        action_id=assigned_action.actionId,
                        action_payload=action_payload,
                        action_type=action_type,
                    )

                    yield action

            except grpc.RpcError as e:
                # Handle different types of errors
                if e.code() == grpc.StatusCode.CANCELLED:
                    # Context cancelled, unsubscribe and close
                    # self.logger.debug("Context cancelled, closing listener")
                    break
                elif e.code() == grpc.StatusCode.UNAVAILABLE:
                    # Retry logic
                    logger.info("Could not connect to Hatchet, retrying...")
                    self.retries = self.retries + 1
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    logger.info("Deadline exceeded, retrying subscription")
                    continue
                else:
                    # Unknown error, report and break
                    # self.logger.error(f"Failed to receive message: {e}")
                    # err_ch(e)
                    logger.error(f"Failed to receive message: {e}")
                    break

    def parse_action_payload(self, payload : str):
        try:
            payload_data = json.loads(payload)
        except json.JSONDecodeError as e:
            raise ValueError(f"Error decoding payload: {e}")
        return payload_data

    def map_action_type(self, action_type):
        if action_type == ActionType.START_STEP_RUN:
            return START_STEP_RUN
        elif action_type == ActionType.CANCEL_STEP_RUN:
            return CANCEL_STEP_RUN
        elif action_type == ActionType.START_GET_GROUP_KEY:
            return START_GET_GROUP_KEY
        else:
            # self.logger.error(f"Unknown action type: {action_type}")
            return None
        
    def get_listen_client(self):
        if self.retries > DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            raise Exception(f"Could not subscribe to the worker after {DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries")
        elif self.retries > 1:
            # logger.info
            # if we are retrying, we wait for a bit. this should eventually be replaced with exp backoff + jitter
            time.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)
        
        return self.client.Listen(WorkerListenRequest(
                    workerId=self.worker_id
                ), 
                timeout=DEFAULT_ACTION_TIMEOUT, 
                metadata=get_metadata(self.token),
            )

    def unregister(self):
        try:
            self.client.Unsubscribe(
                WorkerUnsubscribeRequest(
                    workerId=self.worker_id
                ),
                timeout=DEFAULT_REGISTER_TIMEOUT,
                metadata=get_metadata(self.token),
            )
        except grpc.RpcError as e:
            raise Exception(f"Failed to unsubscribe: {e}")

class DispatcherClientImpl(DispatcherClient):
    def __init__(self, client : DispatcherStub, token):
        self.client = client
        self.token = token
        # self.logger = logger
        # self.validator = validator

    def get_action_listener(self, req: GetActionListenerRequest) -> ActionListenerImpl:
        # Register the worker
        response : WorkerRegisterResponse = self.client.Register(WorkerRegisterRequest(
            workerName=req.worker_name,
            actions=req.actions,
            services=req.services
        ), timeout=DEFAULT_REGISTER_TIMEOUT, metadata=get_metadata(self.token))

        return ActionListenerImpl(self.client, self.token, response.workerId)

    def send_step_action_event(self, in_: StepActionEvent):
        response : ActionEventResponse = self.client.SendStepActionEvent(in_, metadata=get_metadata(self.token),)

        return response
    
    def send_group_key_action_event(self, in_: GroupKeyActionEvent):
        response : ActionEventResponse = self.client.SendGroupKeyActionEvent(in_, metadata=get_metadata(self.token),)

        return response

