import { Timestamp } from '@protoc/google/protobuf/Timestamp';
import { Workflow } from '../../workflow';
import { ClientConfig } from '../client/client-config';

export class AdminClient {
  config: ClientConfig;

  constructor(config: ClientConfig) {
    this.config = config;
  }

  put_workflow(workflow: Workflow, options?: { autoVersion?: boolean }) {
    throw new Error('not implemented');
  }

  schedule_workflow(workflow: Workflow, options?: { schedules?: Timestamp[] }) {
    throw new Error('not implemented');
  }
}
