import { createFetchTransport, resolveServerUrl } from './fetch-transport';

const TOKEN = 'eyJhbGciOi.eyJzdWIi.sig';

describe('resolveServerUrl', () => {
  it.each([
    ['https://e.example', undefined, 'https://e.example'],
    ['https://e.example/', { strategy: 'tls' as const }, 'https://e.example'],
    ['HTTPS://E.example:7070/base///', undefined, 'HTTPS://E.example:7070/base'],
    ['http://e.example:7070', { strategy: 'none' as const }, 'http://e.example:7070'],
    ['http://127.0.0.1:7070/base', { strategy: 'none' as const }, 'http://127.0.0.1:7070/base'],
    ['http://[::1]:7070', { strategy: 'none' as const }, 'http://[::1]:7070'],
  ])('accepts serverUrl %s with tls %j', (serverUrl, tls, expected) => {
    expect(resolveServerUrl({ serverUrl, tls })).toBe(expected);
  });

  it.each([
    ['http://e.example:7070', undefined, /serverUrl is http:\/\/ but tls\.strategy is 'tls'/],
    ['http://e.example:7070', { strategy: 'tls' as const }, /serverUrl is http:\/\/ but tls\.strategy/],
    ['https://e.example', { strategy: 'none' as const }, /serverUrl is https:\/\/ but tls\.strategy is 'none'/],
    ['https://user:secret@e.example', undefined, /serverUrl must not carry a username or password/],
    ['https://e.example/base?x=1', undefined, /serverUrl must not carry a query string or fragment/],
    ['https://e.example/base?', undefined, /serverUrl must not carry a query string or fragment/],
    ['https://e.example/base#frag', undefined, /serverUrl must not carry a query string or fragment/],
    ['ftp://e.example', undefined, /serverUrl must be an absolute http:\/\/ or https:\/\/ URL/],
    ['e.example:7070', undefined, /serverUrl must be an absolute http:\/\/ or https:\/\/ URL/],
    ['//e.example', undefined, /serverUrl must be an absolute http:\/\/ or https:\/\/ URL/],
  ])('refuses serverUrl %s with tls %j', (serverUrl, tls, message) => {
    expect(() => resolveServerUrl({ serverUrl, tls })).toThrow(message);
    expect(() => createFetchTransport({ token: TOKEN, serverUrl, tls })).toThrow(message);
  });

  it('never echoes the URL, so a password in it stays out of the error', () => {
    expect(() => resolveServerUrl({ serverUrl: 'https://user:secret@e.example' })).toThrow(
      expect.not.objectContaining({ message: expect.stringContaining('secret') })
    );
  });

  it.each([
    ['e.example:7070', undefined, 'https://e.example:7070'],
    ['e.example:7070', { strategy: 'tls' as const }, 'https://e.example:7070'],
    ['e.example:7070', { strategy: 'none' as const }, 'http://e.example:7070'],
    ['dns:///e.example', undefined, 'https://e.example:443'],
  ])('derives the scheme for hostPort %s from tls %j', (hostPort, tls, expected) => {
    expect(resolveServerUrl({ hostPort, tls })).toBe(expected);
  });

  it('needs one of serverUrl and hostPort', () => {
    expect(() => resolveServerUrl({})).toThrow(/needs a serverUrl or a hostPort/);
  });
});
