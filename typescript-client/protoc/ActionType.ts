// Original file: ../api-contracts/dispatcher/dispatcher.proto

export const ActionType = {
  START_STEP_RUN: 0,
  CANCEL_STEP_RUN: 1,
} as const;

export type ActionType =
  | 'START_STEP_RUN'
  | 0
  | 'CANCEL_STEP_RUN'
  | 1

export type ActionType__Output = typeof ActionType[keyof typeof ActionType]
