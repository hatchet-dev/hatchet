// Original file: ../api-contracts/workflows/workflows.proto

import type { CreateWorkflowStepOpts as _CreateWorkflowStepOpts, CreateWorkflowStepOpts__Output as _CreateWorkflowStepOpts__Output } from './CreateWorkflowStepOpts';

export interface CreateWorkflowJobOpts {
  'name'?: (string);
  'description'?: (string);
  'timeout'?: (string);
  'steps'?: (_CreateWorkflowStepOpts)[];
}

export interface CreateWorkflowJobOpts__Output {
  'name'?: (string);
  'description'?: (string);
  'timeout'?: (string);
  'steps'?: (_CreateWorkflowStepOpts__Output)[];
}
