# relative imports
import json
import threading
import time
from typing import Callable, List, Union

import grpc

from ..dispatcher_pb2 import (
    ActionEventResponse,
    ActionType,
    AssignedAction,
    GroupKeyActionEvent,
    HeartbeatRequest,
    OverridesData,
    StepActionEvent,
    WorkerListenRequest,
    WorkerRegisterRequest,
    WorkerRegisterResponse,
    WorkerUnsubscribeRequest,
)
from ..dispatcher_pb2_grpc import DispatcherStub
from ..loader import ClientConfig
from ..logger import logger
from ..metadata import get_metadata
from .events import proto_timestamp_now


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


DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 15
DEFAULT_ACTION_TIMEOUT = 600  # seconds
DEFAULT_REGISTER_TIMEOUT = 30


class GetActionListenerRequest:
    def __init__(
        self,
        worker_name: str,
        services: List[str],
        actions: List[str],
        max_runs: int | None = None,
    ):
        self.worker_name = worker_name
        self.services = services
        self.actions = actions
        self.max_runs = max_runs


class Action:
    def __init__(
        self,
        worker_id: str,
        tenant_id: str,
        workflow_run_id: str,
        get_group_key_run_id: str,
        job_id: str,
        job_name: str,
        job_run_id: str,
        step_id: str,
        step_run_id: str,
        action_id: str,
        action_payload: str,
        action_type: ActionType,
    ):
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
    def __init__(self, client: DispatcherStub, token, worker_id):
        self.client = client
        self.token = token
        self.worker_id = worker_id
        self.retries = 0
        self.last_connection_attempt = 0
        self.heartbeat_thread = None
        self.run_heartbeat = True
        self.listen_strategy = "v2"

    def heartbeat(self):
        # send a heartbeat every 4 seconds
        while True:
            if not self.run_heartbeat:
                break

            try:
                self.client.Heartbeat(
                    HeartbeatRequest(
                        workerId=self.worker_id,
                        heartbeatAt=proto_timestamp_now(),
                    ),
                    timeout=DEFAULT_REGISTER_TIMEOUT,
                    metadata=get_metadata(self.token),
                )
            except grpc.RpcError as e:
                # we don't reraise the error here, as we don't want to stop the heartbeat thread
                logger.error(f"Failed to send heartbeat: {e}")

                if e.code() == grpc.StatusCode.UNIMPLEMENTED:
                    break

            time.sleep(4)

    def start_heartbeater(self):
        if self.heartbeat_thread is not None:
            return

        # create a new thread to send heartbeats
        heartbeat_thread = threading.Thread(target=self.heartbeat)
        heartbeat_thread.start()

        self.heartbeat_thread = heartbeat_thread

    def actions(self):
        while True:
            logger.info("Connecting to Hatchet to establish listener for actions...")

            try:
                for assigned_action in self.get_listen_client():
                    self.retries = 0
                    assigned_action: AssignedAction

                    # Process the received action
                    action_type = self.map_action_type(assigned_action.actionType)

                    if (
                        assigned_action.actionPayload is None
                        or assigned_action.actionPayload == ""
                    ):
                        action_payload = None
                    else:
                        action_payload = self.parse_action_payload(
                            assigned_action.actionPayload
                        )

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
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    logger.info("Deadline exceeded, retrying subscription")
                    continue
                elif (
                    self.listen_strategy == "v2"
                    and e.code() == grpc.StatusCode.UNIMPLEMENTED
                ):
                    # ListenV2 is not available, fallback to Listen
                    self.listen_strategy = "v1"
                    self.run_heartbeat = False
                    logger.info("ListenV2 not available, falling back to Listen")
                    continue
                else:
                    # Unknown error, report and break
                    # self.logger.error(f"Failed to receive message: {e}")
                    # err_ch(e)
                    logger.error(f"Failed to receive message: {e}")

                    self.retries = self.retries + 1

    def parse_action_payload(self, payload: str):
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
        current_time = int(time.time())

        if (
            current_time - self.last_connection_attempt
            > DEFAULT_ACTION_LISTENER_RETRY_INTERVAL
        ):
            self.retries = 0

        if self.retries > DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            raise Exception(
                f"Could not subscribe to the worker after {DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries"
            )
        elif self.retries >= 1:
            # logger.info
            # if we are retrying, we wait for a bit. this should eventually be replaced with exp backoff + jitter
            time.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)
            logger.info(
                f"Could not connect to Hatchet, retrying... {self.retries}/{DEFAULT_ACTION_LISTENER_RETRY_COUNT}"
            )

        if self.listen_strategy == "v2":
            listener = self.client.ListenV2(
                WorkerListenRequest(workerId=self.worker_id),
                metadata=get_metadata(self.token),
            )

            self.start_heartbeater()
        else:
            # if ListenV2 is not available, fallback to Listen
            listener = self.client.Listen(
                WorkerListenRequest(workerId=self.worker_id),
                timeout=DEFAULT_ACTION_TIMEOUT,
                metadata=get_metadata(self.token),
            )

        self.last_connection_attempt = current_time

        logger.info("Listener established.")
        return listener

    def unregister(self):
        self.run_heartbeat = False

        try:
            self.client.Unsubscribe(
                WorkerUnsubscribeRequest(workerId=self.worker_id),
                timeout=DEFAULT_REGISTER_TIMEOUT,
                metadata=get_metadata(self.token),
            )
        except grpc.RpcError as e:
            raise Exception(f"Failed to unsubscribe: {e}")


class DispatcherClientImpl(DispatcherClient):
    def __init__(self, client: DispatcherStub, token):
        self.client = client
        self.token = token
        # self.logger = logger
        # self.validator = validator

    def get_action_listener(self, req: GetActionListenerRequest) -> ActionListenerImpl:
        # Register the worker
        response: WorkerRegisterResponse = self.client.Register(
            WorkerRegisterRequest(
                workerName=req.worker_name,
                actions=req.actions,
                services=req.services,
                maxRuns=req.max_runs,
            ),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(self.token),
        )

        return ActionListenerImpl(self.client, self.token, response.workerId)

    def send_step_action_event(self, in_: StepActionEvent):
        response: ActionEventResponse = self.client.SendStepActionEvent(
            in_,
            metadata=get_metadata(self.token),
        )

        return response

    def send_group_key_action_event(self, in_: GroupKeyActionEvent):
        response: ActionEventResponse = self.client.SendGroupKeyActionEvent(
            in_,
            metadata=get_metadata(self.token),
        )

        return response

    def put_overrides_data(self, data: OverridesData):
        response: ActionEventResponse = self.client.PutOverridesData(
            data,
            metadata=get_metadata(self.token),
        )

        return response
