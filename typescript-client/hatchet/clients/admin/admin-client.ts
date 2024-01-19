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

  schedule_workflow(workflow: Workflow, options?: { schedules?: any[] }) {
    throw new Error('not implemented');
  }
}
