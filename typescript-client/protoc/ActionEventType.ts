// Original file: ../api-contracts/dispatcher/dispatcher.proto

export const ActionEventType = {
  STEP_EVENT_TYPE_UNKNOWN: 0,
  STEP_EVENT_TYPE_STARTED: 1,
  STEP_EVENT_TYPE_COMPLETED: 2,
  STEP_EVENT_TYPE_FAILED: 3,
} as const;

export type ActionEventType =
  | 'STEP_EVENT_TYPE_UNKNOWN'
  | 0
  | 'STEP_EVENT_TYPE_STARTED'
  | 1
  | 'STEP_EVENT_TYPE_COMPLETED'
  | 2
  | 'STEP_EVENT_TYPE_FAILED'
  | 3

export type ActionEventType__Output = typeof ActionEventType[keyof typeof ActionEventType]
