import { isHeartbeatMessage } from './heartbeat-controller';

describe('isHeartbeatMessage', () => {
  it('accepts heartbeat log messages', () => {
    expect(isHeartbeatMessage({ type: 'debug', message: 'Heartbeat sent' })).toBe(true);
    expect(isHeartbeatMessage({ type: 'info', message: 'Heartbeat started' })).toBe(true);
    expect(isHeartbeatMessage({ type: 'warn', message: 'Heartbeat delayed' })).toBe(true);
    expect(isHeartbeatMessage({ type: 'error', message: 'Heartbeat failed' })).toBe(true);
  });

  it('ignores Node watch-mode worker dependency messages', () => {
    expect(isHeartbeatMessage({ 'watch:require': ['/worker-dep.js'] })).toBe(false);
    expect(isHeartbeatMessage({ 'watch:import': ['file:///worker-dep.mjs'] })).toBe(false);
  });

  it('rejects malformed heartbeat messages', () => {
    expect(isHeartbeatMessage(undefined)).toBe(false);
    expect(isHeartbeatMessage('debug')).toBe(false);
    expect(isHeartbeatMessage({ type: 'debug' })).toBe(false);
    expect(isHeartbeatMessage({ type: 'green', message: 'not a heartbeat level' })).toBe(false);
    expect(isHeartbeatMessage({ type: 'error', message: new Error('failed') })).toBe(false);
  });
});
