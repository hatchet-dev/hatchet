import { describe, expect, it } from 'vitest';
import { hatchet } from '../index';
import { buildRegistry } from './registry';

const quiet = () => {};

describe('buildRegistry', () => {
  it('serves the generated on-success task over the socket, like an on-failure task', () => {
    const workflow = hatchet.workflow<{ n: number }>({ name: 'with-success' });
    workflow.task({ name: 'first', fn: async () => ({ first: true }) });
    workflow.onSuccess({ fn: async () => ({ done: true }) });

    const registry = buildRegistry([workflow], undefined, quiet);

    expect(registry.runners.has('with-success:on-success-task')).toBe(true);
    expect(registry.streamActions.has('with-success:on-success-task')).toBe(true);
    expect(registry.pureActions.has('with-success:on-success-task')).toBe(false);
    expect(registry.workflows[0].actions).toContain('with-success:on-success-task');
  });

  it('refuses two tasks whose names lowercase to one action id', () => {
    const workflow = hatchet.workflow<{ n: number }>({ name: 'collision' });
    workflow.task({ name: 'Task', hatchetClient: false, fn: async () => ({}) });
    workflow.task({ name: 'task', fn: async () => ({}) });

    expect(() => buildRegistry([workflow], undefined, quiet)).toThrow(
      /"Task" and "task", which share the action id "collision:task"/
    );
  });

  it('refuses an empty serve list, which the healthcheck would read as every task', () => {
    const task = hatchet.task({ name: 'only', fn: async () => ({}) });

    expect(() => buildRegistry([task], [], quiet)).toThrow(/`serve` is empty/);
    expect(buildRegistry([task], [task], quiet).served.has('only:only')).toBe(true);
  });

  it('keeps a pure task pure through the copies the registration is built from', () => {
    const pure = hatchet.task({ name: 'pure', hatchetClient: false, fn: async () => ({}) });
    const social = hatchet.task({ name: 'social', fn: async () => ({}) });
    const workflow = hatchet.workflow<{ n: number }>({ name: 'mixed' });
    workflow.task({ name: 'quiet', hatchetClient: false, fn: async () => ({}) });
    workflow.task({ name: 'loud', fn: async () => ({}) });

    const registry = buildRegistry([pure, social, workflow], undefined, quiet);

    expect([...registry.pureActions].sort()).toEqual(['mixed:quiet', 'pure:pure']);
    expect([...registry.streamActions].sort()).toEqual(['mixed:loud', 'social:social']);
    // The mark lives on the entry, not in a module, and never reaches the registration.
    expect(JSON.stringify(registry.workflows[0].proto)).not.toContain('pure-task');
  });
});
