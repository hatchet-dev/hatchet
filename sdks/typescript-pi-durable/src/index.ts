import '@hatchet-dev/sandbox';

export { SessionClient, type SessionClientOptions } from './client.js';
export {
  recordingSnapshot,
  SandboxDoc,
  type SandboxCodingOptions,
  type SandboxState,
  sandboxCoding,
  sandboxEnvs,
} from './extension.js';
export {
  ClientMessage,
  type ConversationSummary,
  InboxMessage,
  type ItemView,
  type LiveToolView,
  type RejectedMessage,
  SessionId,
  type SessionView,
  sessionTopics,
  type ToolCallView,
} from './protocol.js';
export {
  DEFAULT_WORKSPACE,
  latestSnapshot,
  SandboxEnv,
  type SandboxEnvOptions,
  sandboxHandle,
} from './sandbox-env.js';
export {
  defineSessionTask,
  HostDoc,
  type SessionTaskInput,
  type SessionTaskOptions,
  type SessionTaskOutput,
} from './session-host.js';
export { summarize } from './view.js';
