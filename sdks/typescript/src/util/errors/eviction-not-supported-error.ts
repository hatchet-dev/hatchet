import { MinEngineVersion } from '@hatchet/v1';
import HatchetError from './hatchet-error';

export class EvictionNotSupportedError extends HatchetError {
  engineVersion: string | undefined;

  constructor(engineVersion?: string) {
    const versionInfo = engineVersion ? ` (engine ${engineVersion})` : '';
    super(
      `Durable eviction is not supported by the connected engine${versionInfo}. ` +
        `Please upgrade your Hatchet engine to ${MinEngineVersion.DURABLE_EVICTION} or later.`
    );
    this.name = 'EvictionNotSupportedError';
    this.engineVersion = engineVersion;
  }
}
