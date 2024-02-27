import { DispatcherClient as PbDispatcherClient, AssignedAction } from '@hatchet/protoc/dispatcher';

import { Status } from 'nice-grpc';
import { ClientConfig } from '@clients/hatchet-client/client-config';
import sleep from '@util/sleep';
import HatchetError from '@util/errors/hatchet-error';
import { Logger } from '@hatchet/util/logger';

import { DispatcherClient } from './dispatcher-client';

const DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5; // seconds
const DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5;

export interface Action {
  tenantId: string;
  jobId: string;
  jobName: string;
  jobRunId: string;
  stepId: string;
  stepRunId: string;
  actionId: string;
  actionType: number;
  actionPayload: string;
  workflowRunId: string;
  getGroupKeyRunId: string;
}

export class ActionListener {
  config: ClientConfig;
  client: PbDispatcherClient;
  listener: AsyncIterable<AssignedAction>;
  workerId: string;
  logger: Logger;
  lastConnectionAttempt: number = 0;
  retries: number = 0;

  constructor(client: DispatcherClient, listener: AsyncIterable<AssignedAction>, workerId: string) {
    this.config = client.config;
    this.client = client.client;
    this.listener = listener;
    this.workerId = workerId;
    this.logger = new Logger(`ActionListener`, this.config.log_level);
  }

  actions = () =>
    (async function* gen(client: ActionListener) {
      while (true) {
        try {
          for await (const assignedAction of await client.getListenClient()) {
            const action: Action = {
              ...assignedAction,
            };

            yield action;
          }
        } catch (e: any) {
          // if this is a HatchetError, we should throw this error
          if (e instanceof HatchetError) {
            throw e;
          }

          if (e.code === Status.CANCELLED) {
            break;
          }

          client.incrementRetries();
        }
      }
    })(this);

  async incrementRetries() {
    this.retries += 1;
  }

  async getListenClient(): Promise<AsyncIterable<AssignedAction>> {
    const currentTime = Math.floor(Date.now() / 1000);

    // subtract 1 from the last connection attempt to account for the time it takes to establish the listener
    if (currentTime - this.lastConnectionAttempt - 1 > DEFAULT_ACTION_LISTENER_RETRY_INTERVAL) {
      this.retries = 0;
    }

    this.lastConnectionAttempt = currentTime;

    if (this.retries > DEFAULT_ACTION_LISTENER_RETRY_COUNT) {
      throw new HatchetError(
        `Could not subscribe to the worker after ${DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries`
      );
    }

    this.logger.info(
      `Connecting to Hatchet to establish listener for actions... ${this.retries}/${DEFAULT_ACTION_LISTENER_RETRY_COUNT} (last attempt: ${this.lastConnectionAttempt})`
    );

    if (this.retries >= 1) {
      await sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL * 1000);
    }

    try {
      this.listener = this.client.listen({
        workerId: this.workerId,
      });

      return this.listener;
    } catch (e: any) {
      this.retries += 1;
      this.logger.error(`Attempt ${this.retries}: Failed to connect, retrying...`); // Optional: log retry attempt

      return this.getListenClient();
    }
  }

  async unregister() {
    try {
      return this.client.unsubscribe({
        workerId: this.workerId,
      });
    } catch (e: any) {
      throw new HatchetError(`Failed to unsubscribe: ${e.message}`);
    }
  }
}

// def get_listen_client(self):
//         current_time = int(time.time())

//         if current_time-self.last_connection_attempt > DEFAULT_ACTION_LISTENER_RETRY_INTERVAL:
//             self.retries = 0

//         if self.retries > DEFAULT_ACTION_LISTENER_RETRY_COUNT:
//             raise Exception(
//                 f"Could not subscribe to the worker after {DEFAULT_ACTION_LISTENER_RETRY_COUNT} retries")
//         elif self.retries >= 1:
//             # logger.info
//             # if we are retrying, we wait for a bit. this should eventually be replaced with exp backoff + jitter
//             time.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)
//             logger.info(
//                 f"Could not connect to Hatchet, retrying... {self.retries}/{DEFAULT_ACTION_LISTENER_RETRY_COUNT}")

//         listener = self.client.Listen(WorkerListenRequest(
//             workerId=self.worker_id
//         ),
//             timeout=DEFAULT_ACTION_TIMEOUT,
//             metadata=get_metadata(self.token),
//         )

//         self.last_connection_attempt = current_time

//         logger.info('Listener established.')
//         return listener

// def actions(self):
//         while True:
//             logger.info(
//                 "Connecting to Hatchet to establish listener for actions...")

//             try:
//                 for assigned_action in self.get_listen_client():
//                     self.retries = 0
//                     assigned_action: AssignedAction

//                     # Process the received action
//                     action_type = self.map_action_type(assigned_action.actionType)

//                     if assigned_action.actionPayload is None or assigned_action.actionPayload == "":
//                         action_payload = None
//                     else:
//                         action_payload = self.parse_action_payload(assigned_action.actionPayload)

//                     action = Action(
//                         tenant_id=assigned_action.tenantId,
//                         worker_id=self.worker_id,
//                         workflow_run_id=assigned_action.workflowRunId,
//                         get_group_key_run_id=assigned_action.getGroupKeyRunId,
//                         job_id=assigned_action.jobId,
//                         job_name=assigned_action.jobName,
//                         job_run_id=assigned_action.jobRunId,
//                         step_id=assigned_action.stepId,
//                         step_run_id=assigned_action.stepRunId,
//                         action_id=assigned_action.actionId,
//                         action_payload=action_payload,
//                         action_type=action_type,
//                     )

//                     yield action

//             except grpc.RpcError as e:
//                 # Handle different types of errors
//                 if e.code() == grpc.StatusCode.CANCELLED:
//                     # Context cancelled, unsubscribe and close
//                     # self.logger.debug("Context cancelled, closing listener")
//                     break
//                 elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
//                     logger.info("Deadline exceeded, retrying subscription")
//                     continue
//                 else:
//                     # Unknown error, report and break
//                     # self.logger.error(f"Failed to receive message: {e}")
//                     # err_ch(e)
//                     logger.error(f"Failed to receive message: {e}")

//                     self.retries = self.retries + 1
