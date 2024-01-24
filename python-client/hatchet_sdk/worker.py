import json
import signal
import sys

import grpc
from typing import Any, Callable, Dict
from .workflow import WorkflowMeta
from .clients.dispatcher import GetActionListenerRequest, ActionListenerImpl, Action
from .dispatcher_pb2 import STEP_EVENT_TYPE_FAILED, ActionType, ActionEvent, ActionEventType, STEP_EVENT_TYPE_COMPLETED, STEP_EVENT_TYPE_STARTED
from .client import new_client 
from concurrent.futures import ThreadPoolExecutor, Future
from google.protobuf.timestamp_pb2 import Timestamp
from .context import Context
from .logger import logger

# Worker class
class Worker:
    def __init__(self, name: str, max_threads: int = 200, debug=False, handle_kill=True):
        self.name = name
        self.thread_pool = ThreadPoolExecutor(max_workers=max_threads)
        self.futures: Dict[str, Future] = {}  # Store step run ids and futures
        self.action_registry : dict[str, Callable[..., Any]] = {} 

        signal.signal(signal.SIGINT, self.exit_gracefully)
        signal.signal(signal.SIGTERM, self.exit_gracefully)

        self.killing = False
        self.handle_kill = handle_kill

    def handle_start_step_run(self, action : Action):
        action_name = action.action_id  # Assuming action object has 'name' attribute
        context = Context(action.action_payload)  # Assuming action object has 'context' attribute

        # Find the corresponding action function from the registry
        action_func = self.action_registry.get(action_name)

        if action_func:
            def callback(future : Future):
                errored = False

                # Get the output from the future
                try:
                    output = future.result()
                except Exception as e:
                    errored = True

                    # This except is coming from the application itself, so we want to send that to the Hatchet instance
                    event = self.get_action_event(action, STEP_EVENT_TYPE_FAILED)
                    event.eventPayload = str(e)

                    try:
                        self.client.dispatcher.send_action_event(event)
                    except Exception as e:
                        logger.error(f"Could not send action event: {e}")

                if not errored:
                    # Create an action event
                    try:
                        event = self.get_action_finished_event(action, output)
                    except Exception as e:
                        logger.error(f"Could not get action finished event: {e}")
                        raise e

                    # Send the action event to the dispatcher
                    self.client.dispatcher.send_action_event(event)

                # Remove the future from the dictionary
                del self.futures[action.step_run_id]
                del self.futures[action.step_run_id + "_callback"]

            # Submit the action to the thread pool
            future = self.thread_pool.submit(action_func, context)
            callback = self.thread_pool.submit(callback, future)
            self.futures[action.step_run_id] = future
            self.futures[action.step_run_id + "_callback"] = callback

            # send an event that the step run has started
            try:
                event = self.get_action_event(action, STEP_EVENT_TYPE_STARTED)
            except Exception as e:
                logger.error(f"Could not create action event: {e}")

            # Send the action event to the dispatcher
            self.client.dispatcher.send_action_event(event)
                
    def handle_cancel_step_run(self, action : Action):
        step_run_id = action.step_run_id

        future = self.futures.get(step_run_id)

        if future:
            future.cancel()
            del self.futures[step_run_id]
    
    def get_action_event(self, action : Action, event_type : ActionEventType) -> ActionEvent:
        # timestamp 
        # eventTimestamp = datetime.datetime.now(datetime.timezone.utc)
        # eventTimestamp = eventTimestamp.isoformat() 
        eventTimestamp = Timestamp()
        eventTimestamp.GetCurrentTime()

        return ActionEvent(
            tenantId=action.tenant_id,
            workerId=action.worker_id,
            jobId=action.job_id,
            jobRunId=action.job_run_id,
            stepId=action.step_id,
            stepRunId=action.step_run_id,
            actionId=action.action_id,
            eventTimestamp=eventTimestamp,
            eventType=event_type,
        )
    
    def get_action_finished_event(self, action : Action, output : Any) -> ActionEvent:
        try:
            event = self.get_action_event(action, STEP_EVENT_TYPE_COMPLETED)
        except Exception as e:
            logger.error(f"Could not create action finished event: {e}")
            raise e

        output_bytes = ''

        if output is not None:
            output_bytes = json.dumps(output)

        event.eventPayload = output_bytes

        return event
    
    def register_workflow(self, workflow : WorkflowMeta):
        def create_action_function(action_func):
            def action_function(context):
                return action_func(workflow, context)
            return action_function

        for action_name, action_func in workflow.get_actions():
            self.action_registry[action_name] = create_action_function(action_func)

    def exit_gracefully(self, signum, frame):
        self.killing = True

        logger.info("Gracefully exiting hatchet worker...")

        try:
            self.listener.unregister()
        except Exception as e:
            logger.error(f"Could not unregister worker: {e}")

        # wait for futures to complete
        for future in self.futures.values():
            try:
                future.result()
            except Exception as e:
                logger.error(f"Could not wait for future: {e}")

        if self.handle_kill:
            logger.info("Exiting...")
            sys.exit(0)
    
    def start(self, retry_count=1):
        logger.info("Starting worker...")

        self.client = new_client()

        try:
            self.listener : ActionListenerImpl = self.client.dispatcher.get_action_listener(GetActionListenerRequest(
                worker_name=self.name,
                services=["default"],
                actions=self.action_registry.keys(),
            ))

            generator = self.listener.actions()

            for action in generator:
                if action.action_type == ActionType.START_STEP_RUN:
                    self.handle_start_step_run(action)
                elif action.action_type == ActionType.CANCEL_STEP_RUN:
                    self.handle_cancel_step_run(action)

                pass  # Replace this with your actual processing code
        except grpc.RpcError as rpc_error:
            logger.error(f"Could not start worker: {rpc_error}")

        # if we are here, but not killing, then we should retry start
        if not self.killing:
            if retry_count > 5:
                raise Exception("Could not start worker after 5 retries")
            
            logger.info("Could not start worker, retrying...")
            
            self.start(retry_count + 1)