// Original file: ../api-contracts/workflows/workflows.proto

import type { CreateWorkflowVersionOpts as _CreateWorkflowVersionOpts, CreateWorkflowVersionOpts__Output as _CreateWorkflowVersionOpts__Output } from './CreateWorkflowVersionOpts';

export interface PutWorkflowRequest {
  'tenantId'?: (string);
  'opts'?: (_CreateWorkflowVersionOpts | null);
}

export interface PutWorkflowRequest__Output {
  'tenantId'?: (string);
  'opts'?: (_CreateWorkflowVersionOpts__Output);
}
