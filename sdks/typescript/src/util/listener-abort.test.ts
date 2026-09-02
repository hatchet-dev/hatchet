import { AbortError } from '@hatchet/util/abort-error';
import {
  isListenerReconnectAbort,
  isListenerShutdownAbort,
  LISTENER_RECONNECT_REASON,
  LISTENER_SHUTDOWN_REASON,
} from '../clients/dispatcher/listener-abort';

describe('listener abort helpers', () => {
  it('detects reconnect abort reason', () => {
    expect(isListenerReconnectAbort(new AbortError(LISTENER_RECONNECT_REASON))).toBe(true);
    expect(isListenerReconnectAbort(new AbortError(LISTENER_SHUTDOWN_REASON))).toBe(false);
  });

  it('detects shutdown abort reason', () => {
    expect(isListenerShutdownAbort(new AbortError(LISTENER_SHUTDOWN_REASON))).toBe(true);
    expect(isListenerShutdownAbort(new AbortError(LISTENER_RECONNECT_REASON))).toBe(false);
  });
});
