# relative imports
from ..dispatcher_pb2 import ActionEvent, ActionEventResponse, ActionType, AssignedAction, WorkerListenRequest, WorkerRegisterRequest, WorkerUnsubscribeRequest, WorkerRegisterResponse
from ..dispatcher_pb2_grpc import DispatcherStub

import time
from ..loader import ClientConfig
import json
import grpc
from typing import Callable, List, Union

def new_dispatcher(conn, config: ClientConfig):
    return DispatcherClientImpl(
        client=DispatcherStub(conn),
        tenant_id=config.tenant_id,
        # logger=shared_opts['logger'],
        # validator=shared_opts['validator'],
    )
    
class DispatcherClient:
    def get_action_listener(self, ctx, req):
        raise NotImplementedError

    def send_action_event(self, ctx, in_):
        raise NotImplementedError

DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5

class GetActionListenerRequest:
    def __init__(self, worker_name: str, services: List[str], actions: List[str]):
        self.worker_name = worker_name
        self.services = services
        self.actions = actions

class Action:
    def __init__(self, worker_id: str, tenant_id: str, job_id: str, job_name: str, job_run_id: str, step_id: str, step_run_id: str, action_id: str, action_payload: str, action_type: ActionType):
        self.worker_id = worker_id
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

# enum for START_STEP_RUN and CANCEL_STEP_RUN
START_STEP_RUN = 0
CANCEL_STEP_RUN = 1

class ActionListenerImpl(WorkerActionListener):
    def __init__(self, client : DispatcherStub, tenant_id, listen_client, worker_id):
        self.client = client
        self.tenant_id = tenant_id
        self.listen_client = listen_client
        self.worker_id = worker_id
        # self.logger = logger
        # self.validator = validator

    def actions(self):
        while True:
            try:
                for assigned_action in self.listen_client:
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
                    self.retry_subscribe()
                else:
                    # Unknown error, report and break
                    # self.logger.error(f"Failed to receive message: {e}")
                    # err_ch(e)
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
        else:
            # self.logger.error(f"Unknown action type: {action_type}")
            return None

    def retry_subscribe(self):
        retries = 0
        while retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            try:
                time.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)
                self.listen_client = self.client.Listen(WorkerListenRequest(
                    tenantId=self.tenant_id,
                    workerId=self.worker_id
                ))
                return
            except grpc.RpcError as e:
                retries += 1
                # self.logger.error(f"Failed to retry subscription: {e}")

        raise Exception(f"Could not subscribe to the worker after {DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries")

    def unregister(self):
        try:
            self.client.Unsubscribe(
                WorkerUnsubscribeRequest(
                    tenant_id=self.tenant_id,
                    worker_id=self.worker_id
                )
            )
        except grpc.RpcError as e:
            raise Exception(f"Failed to unsubscribe: {e}")

class DispatcherClientImpl(DispatcherClient):
    def __init__(self, client : DispatcherStub, tenant_id):
        self.client = client
        self.tenant_id = tenant_id
        # self.logger = logger
        # self.validator = validator

    def get_action_listener(self, req: GetActionListenerRequest) -> ActionListenerImpl:
        # Register the worker
        response : WorkerRegisterResponse = self.client.Register(WorkerRegisterRequest(
            tenantId=self.tenant_id,
            workerName=req.worker_name,
            actions=req.actions,
            services=req.services
        ))

        # Subscribe to the worker
        listener = self.client.Listen(WorkerListenRequest(
            tenantId=self.tenant_id,
            workerId=response.workerId,
        ))

        return ActionListenerImpl(self.client, self.tenant_id, listener, response.workerId)

    def send_action_event(self, in_: ActionEvent):
        response : ActionEventResponse = self.client.SendActionEvent(in_)

        return response

