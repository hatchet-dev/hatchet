import ctypes
import json
import signal
import sys
from threading import Thread, current_thread
import threading
import time

import grpc
from typing import Any, Callable, Dict
from .workflow import WorkflowMeta
from .clients.dispatcher import GetActionListenerRequest, ActionListenerImpl, Action
from .dispatcher_pb2 import ActionType, StepActionEvent, StepActionEventType, GroupKeyActionEvent, GroupKeyActionEventType, STEP_EVENT_TYPE_COMPLETED, STEP_EVENT_TYPE_STARTED, STEP_EVENT_TYPE_FAILED, GROUP_KEY_EVENT_TYPE_STARTED, GROUP_KEY_EVENT_TYPE_COMPLETED, GROUP_KEY_EVENT_TYPE_FAILED
from .client import new_client 
from concurrent.futures import ThreadPoolExecutor, Future
from google.protobuf.timestamp_pb2 import Timestamp
from .context import Context
from .logger import logger

# Worker class
class Worker:
    def __init__(self, name: str, max_threads: int = 200, debug=False, handle_kill=True):
        self.name = name
        self.threads: Dict[str, Thread] = {}  # Store step run ids and threads
        self.thread_pool = ThreadPoolExecutor(max_workers=max_threads)
        self.futures: Dict[str, Future] = {}  # Store step run ids and futures
        self.contexts: Dict[str, Context] = {}  # Store step run ids and contexts
        self.action_registry : dict[str, Callable[..., Any]] = {} 

        signal.signal(signal.SIGINT, self.exit_gracefully)
        signal.signal(signal.SIGTERM, self.exit_gracefully)

        self.killing = False
        self.handle_kill = handle_kill

    def handle_start_step_run(self, action : Action):
        action_name = action.action_id  # Assuming action object has 'name' attribute
        context = Context(action.action_payload)  # Assuming action object has 'context' attribute

        self.contexts[action.step_run_id] = context

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
                    event = self.get_step_action_event(action, STEP_EVENT_TYPE_FAILED)
                    event.eventPayload = str(e)

                    try:
                        self.client.dispatcher.send_step_action_event(event)
                    except Exception as e:
                        logger.error(f"Could not send action event: {e}")

                if not errored:
                    # Create an action event
                    try:
                        event = self.get_step_action_finished_event(action, output)
                    except Exception as e:
                        logger.error(f"Could not get action finished event: {e}")
                        raise e

                    # Send the action event to the dispatcher
                    self.client.dispatcher.send_step_action_event(event)

                # Remove the future from the dictionary
                if action.step_run_id in self.futures:
                    del self.futures[action.step_run_id]

            # Submit the action to the thread pool
            def wrapped_action_func(context):
                # store the thread id
                self.threads[action.step_run_id] = current_thread()

                try:
                    res = action_func(context)
                    return res
                except Exception as e:
                    logger.error(f"Could not execute action: {e}")
                    raise e
                finally:
                    if action.step_run_id in self.threads:
                        # remove the thread id
                        logger.debug(f"Removing step run id {action.step_run_id} from threads")

                        del self.threads[action.step_run_id]

            future = self.thread_pool.submit(wrapped_action_func, context)
            future.add_done_callback(callback)
            self.futures[action.step_run_id] = future

            # send an event that the step run has started
            try:
                event = self.get_step_action_event(action, STEP_EVENT_TYPE_STARTED)
            except Exception as e:
                logger.error(f"Could not create action event: {e}")

            # Send the action event to the dispatcher
            self.client.dispatcher.send_step_action_event(event)

    def handle_start_group_key_run(self, action : Action):
        action_name = action.action_id  # Assuming action object has 'name' attribute
        context = Context(action.action_payload)  # Assuming action object has 'context' attribute

        self.contexts[action.get_group_key_run_id] = context

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
                    event = self.get_group_key_action_event(action, GROUP_KEY_EVENT_TYPE_FAILED)
                    event.eventPayload = str(e)

                    try:
                        self.client.dispatcher.send_group_key_action_event(event)
                    except Exception as e:
                        logger.error(f"Could not send action event: {e}")

                if not errored:
                    # Create an action event
                    try:
                        event = self.get_group_key_action_finished_event(action, output)
                    except Exception as e:
                        logger.error(f"Could not get action finished event: {e}")
                        raise e

                    # Send the action event to the dispatcher
                    self.client.dispatcher.send_group_key_action_event(event)

                # Remove the future from the dictionary
                if action.get_group_key_run_id in self.futures:
                    del self.futures[action.get_group_key_run_id]

            # Submit the action to the thread pool
            def wrapped_action_func(context):
                # store the thread id
                self.threads[action.get_group_key_run_id] = current_thread()

                try:
                    res = action_func(context)
                    return res
                except Exception as e:
                    logger.error(f"Could not execute action: {e}")
                    raise e
                finally:
                    if action.get_group_key_run_id in self.threads:
                        # remove the thread id
                        logger.debug(f"Removing step run id {action.get_group_key_run_id} from threads")

                        del self.threads[action.get_group_key_run_id]

            future = self.thread_pool.submit(wrapped_action_func, context)
            future.add_done_callback(callback)
            self.futures[action.get_group_key_run_id] = future

            # send an event that the step run has started
            try:
                event = self.get_group_key_action_event(action, GROUP_KEY_EVENT_TYPE_STARTED)
            except Exception as e:
                logger.error(f"Could not create action event: {e}")

            # Send the action event to the dispatcher
            self.client.dispatcher.send_group_key_action_event(event)

    def force_kill_thread(self, thread):
        """Terminate a python threading.Thread."""
        try:
            if not thread.is_alive():
                return
            
            logger.info(f"Forcefully terminating thread {thread.ident}")

            exc = ctypes.py_object(SystemExit)
            res = ctypes.pythonapi.PyThreadState_SetAsyncExc(
                ctypes.c_long(thread.ident), exc
            )
            if res == 0:
                raise ValueError("Invalid thread ID")
            elif res != 1:
                logger.error("PyThreadState_SetAsyncExc failed")

                # Call with exception set to 0 is needed to cleanup properly.
                ctypes.pythonapi.PyThreadState_SetAsyncExc(thread.ident, 0)
                raise SystemError("PyThreadState_SetAsyncExc failed")
            
            logger.info(f"Successfully terminated thread {thread.ident}")

            # Immediately add a new thread to the thread pool, because we've actually killed a worker
            # in the ThreadPoolExecutor
            self.thread_pool.submit(lambda: None)
        except Exception as e:
            logger.exception(f"Failed to terminate thread: {e}")
                
    def handle_cancel_action(self, run_id: str):
        # call cancel to signal the context to stop
        context = self.contexts.get(run_id)
        context.cancel()

        future = self.futures.get(run_id)

        if future:
            future.cancel()

            if run_id in self.futures:
                del self.futures[run_id]

        # grace period of 1 second
        time.sleep(1)

        # check if thread is still running, if so, kill it
        if run_id in self.threads:
            thread = self.threads[run_id]

            if thread:
                self.force_kill_thread(thread)

                if run_id in self.threads:
                    del self.threads[run_id]
    
    def get_step_action_event(self, action : Action, event_type : StepActionEventType) -> StepActionEvent:
        eventTimestamp = Timestamp()
        eventTimestamp.GetCurrentTime()

        return StepActionEvent(
            workerId=action.worker_id,
            jobId=action.job_id,
            jobRunId=action.job_run_id,
            stepId=action.step_id,
            stepRunId=action.step_run_id,
            actionId=action.action_id,
            eventTimestamp=eventTimestamp,
            eventType=event_type,
        )
    
    def get_step_action_finished_event(self, action : Action, output : Any) -> StepActionEvent:
        try:
            event = self.get_step_action_event(action, STEP_EVENT_TYPE_COMPLETED)
        except Exception as e:
            logger.error(f"Could not create action finished event: {e}")
            raise e

        output_bytes = ''

        if output is not None:
            output_bytes = json.dumps(output)

        event.eventPayload = output_bytes

        return event
    
    def get_group_key_action_event(self, action : Action, event_type : GroupKeyActionEventType) -> GroupKeyActionEvent:
        eventTimestamp = Timestamp()
        eventTimestamp.GetCurrentTime()

        return GroupKeyActionEvent(
            workerId=action.worker_id,
            workflowRunId=action.workflow_run_id,
            getGroupKeyRunId=action.get_group_key_run_id,
            actionId=action.action_id,
            eventTimestamp=eventTimestamp,
            eventType=event_type,
        )
    
    def get_group_key_action_finished_event(self, action : Action, output : str) -> StepActionEvent:
        try:
            event = self.get_group_key_action_event(action, GROUP_KEY_EVENT_TYPE_COMPLETED)
        except Exception as e:
            logger.error(f"Could not create action finished event: {e}")
            raise e

        event.eventPayload = output

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

        # cancel all futures
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
                    self.thread_pool.submit(self.handle_cancel_action, action.step_run_id)
                elif action.action_type == ActionType.START_GET_GROUP_KEY:
                    self.handle_start_group_key_run(action)
                else:
                    logger.error(f"Unknown action type: {action.action_type}")
        except grpc.RpcError as rpc_error:
            logger.error(f"Could not start worker: {rpc_error}")

        # if we are here, but not killing, then we should retry start
        if not self.killing:
            if retry_count > 5:
                raise Exception("Could not start worker after 5 retries")
            
            logger.info("Could not start worker, retrying...")
            
            self.start(retry_count + 1)