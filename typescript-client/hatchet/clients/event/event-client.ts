import { ClientConfig } from '../client/client-config';

export class EventClient {
  config: ClientConfig;

  constructor(config: ClientConfig) {
    this.config = config;
  }

  push<T>(type: string, input: T) {
    throw new Error('not implemented');
  }
}
