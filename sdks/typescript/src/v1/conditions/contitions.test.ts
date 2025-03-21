import { Action, Condition } from './base';
import {
  SleepCondition,
  UserEventCondition,
  Render,
  Or,
  OrCondition,
  Sleep,
  UserEvent,
} from './index';

export function render(condition: Condition | OrCondition): string {
  if (condition instanceof SleepCondition) {
    return `sleepFor: ${condition.sleepFor}`;
  }
  if (condition instanceof UserEventCondition) {
    return `event: ${condition.eventKey}${condition.expression ? `, expression: ${condition.expression}` : ''}`;
  }
  if (condition instanceof OrCondition) {
    return `OR(${condition.conditions.map((c) => render(c)).join(' || ')})`;
  }
  return 'Unknown condition';
}

describe('Conditions', () => {
  // Basic condition creation tests
  it('should be able to create a sleep condition', () => {
    const condition = new SleepCondition(10);
    expect(condition).toBeDefined();
    expect(condition.sleepFor).toBe(10);
  });

  it('should be able to create a user event condition', () => {
    const condition = new UserEventCondition('user:create', 'user.id == 123');
    expect(condition).toBeDefined();
    expect(condition.eventKey).toBe('user:create');
    expect(condition.expression).toBe('user.id == 123');
  });

  // Wait function tests
  it('should create conditions from object syntax using Wait()', () => {
    const sleep: Sleep = { sleepFor: 15 };
    const userEvent: UserEvent = { eventKey: 'user:update', expression: 'user.status == "active"' };

    const conditions = Render(Action.CREATE, [sleep, userEvent]);

    expect(conditions.length).toBe(2);
    expect(conditions[0]).toBeInstanceOf(SleepCondition);
    expect((conditions[0] as SleepCondition).sleepFor).toBe(15);
    expect(conditions[1]).toBeInstanceOf(UserEventCondition);
    expect((conditions[1] as UserEventCondition).eventKey).toBe('user:update');
    expect((conditions[1] as UserEventCondition).expression).toBe('user.status == "active"');
  });

  it('should accept condition instances in Wait()', () => {
    const sleepCondition = new SleepCondition(5);
    const userEvent: UserEvent = { eventKey: 'user:login' };
    const conditions = Render(Action.CREATE, [sleepCondition, userEvent]);

    expect(conditions.length).toBe(2);
    expect(conditions[0]).toBe(sleepCondition);
    expect(conditions[1]).toBeInstanceOf(UserEventCondition);
  });

  // Or function tests
  it('should create an OR condition using Or()', () => {
    const sleep: Sleep = { sleepFor: 10 };
    const userEvent: UserEvent = { eventKey: 'user:create' };

    const orCondition = Or(sleep, userEvent);

    expect(orCondition).toBeInstanceOf(OrCondition);
    expect(orCondition.conditions.length).toBe(2);

    // Check that both conditions have the same orGroupId
    const { orGroupId } = orCondition.conditions[0].base;
    expect(orGroupId).toBeTruthy();
    expect(orCondition.conditions[1].base.orGroupId).toBe(orGroupId);
  });

  it('should accept condition instances in Or()', () => {
    const sleepCondition = new SleepCondition(7);
    const eventCondition = new UserEventCondition('user:delete', '');

    const orCondition = Or(sleepCondition, eventCondition);

    expect(orCondition.conditions.length).toBe(2);
    expect(orCondition.conditions[0]).toBe(sleepCondition);
    expect(orCondition.conditions[1]).toBe(eventCondition);

    // Check that both conditions have the same orGroupId
    const { orGroupId } = orCondition.conditions[0].base;
    expect(orGroupId).toBeTruthy();
    expect(orCondition.conditions[1].base.orGroupId).toBe(orGroupId);
  });

  // Nested conditions tests
  it('should support Wait(Or()) nested composition', () => {
    const sleep1: Sleep = { sleepFor: 10 };
    const userEvent: UserEvent = { eventKey: 'user:create' };
    const sleep2: Sleep = { sleepFor: 5 };

    const conditions = Render(Action.CREATE, [Or(sleep1, userEvent), sleep2]);

    expect(conditions.length).toBe(3);
    expect(conditions[0]).toBeInstanceOf(SleepCondition);
    expect(conditions[1]).toBeInstanceOf(UserEventCondition);
    expect(conditions[2]).toBeInstanceOf(SleepCondition);

    // Check that OR conditions have the same orGroupId
    const { orGroupId } = conditions[0].base;
    expect(orGroupId).toBeTruthy();
    expect(conditions[1].base.orGroupId).toBe(orGroupId);

    // The second condition should not have the same orGroupId
    const { orGroupId: orGroupId2 } = conditions[1].base;
    expect(orGroupId2).toBeTruthy();
    expect(conditions[2].base.orGroupId).not.toBe(orGroupId2);
  });

  // Render function tests
  it('should render conditions as readable strings', () => {
    const sleepCondition = new SleepCondition(20);
    const eventCondition = new UserEventCondition('user:update', 'user.role == "admin"');

    const sleep: Sleep = { sleepFor: 10 };
    const userEvent: UserEvent = { eventKey: 'user:create', expression: 'user.email != null' };
    const orCondition = Or(sleep, userEvent);

    expect(render(sleepCondition)).toBe('sleepFor: 20');
    expect(render(eventCondition)).toBe('event: user:update, expression: user.role == "admin"');
    expect(render(orCondition)).toMatch(
      /OR\(sleepFor: 10 \|\| event: user:create, expression: user.email != null\)/
    );
  });

  it('should handle empty expressions in render', () => {
    const eventCondition = new UserEventCondition('user:login', '');
    expect(render(eventCondition)).toBe('event: user:login');
  });

  // Error cases
  it('should throw error for invalid condition objects', () => {
    const invalidCondition = { foo: 'bar' };
    expect(() => Render(Action.CREATE, invalidCondition as any)).toThrow();
  });

  // Edge cases
  it('should handle zero sleep duration', () => {
    const sleep: Sleep = { sleepFor: 0 };
    const condition = Render(Action.CREATE, sleep)[0] as SleepCondition;
    expect(condition.sleepFor).toBe(0);
  });

  it('should handle negative sleep duration', () => {
    const sleep: Sleep = { sleepFor: -1 };
    const condition = Render(Action.CREATE, sleep)[0] as SleepCondition;
    expect(condition.sleepFor).toBe(-1);
  });

  // Mixed AND/OR combinations
  it('should handle complex AND/OR combinations', () => {
    const sleep1: Sleep = { sleepFor: 5 };
    const sleep2: Sleep = { sleepFor: 10 };
    const userEvent1: UserEvent = { eventKey: 'user:create' };
    const userEvent2: UserEvent = { eventKey: 'user:update' };
    const userEvent3: UserEvent = { eventKey: 'user:delete' };

    const conditions = Render(Action.CREATE, [
      Or(sleep1, sleep2),
      userEvent1,
      Or(userEvent2, userEvent3),
    ]);

    expect(conditions.length).toBe(5);
    // First two conditions should share an orGroupId
    expect(conditions[0].base.orGroupId).toBe(conditions[1].base.orGroupId);
    // Third condition should have no orGroupId
    expect(conditions[2].base.orGroupId).toBe('');
    // Fourth condition should have a different orGroupId
    expect(conditions[3].base.orGroupId).not.toBe(conditions[0].base.orGroupId);
    expect(conditions[3].base.orGroupId).not.toBe('');
    expect(conditions[3].base.orGroupId).not.toBe(conditions[2].base.orGroupId);
    // Fifth condition should have a different orGroupId
    expect(conditions[4].base.orGroupId).not.toBe(conditions[0].base.orGroupId);
    expect(conditions[4].base.orGroupId).not.toBe('');
    expect(conditions[4].base.orGroupId).not.toBe(conditions[2].base.orGroupId);
  });

  it('should handle complex AND/OR combinations', () => {
    const conditions = Render(Action.CREATE, [
      Or({ sleepFor: 5 }, { eventKey: 'user:update' }),
      { eventKey: 'user:create' },
      Or({ eventKey: 'user:update' }, { eventKey: 'user:delete' }),
    ]);

    expect(conditions.length).toBe(5);

    expect(conditions[0].base.action).toBe(Action.CREATE);
    expect(conditions[1].base.action).toBe(Action.CREATE);
    expect(conditions[2].base.action).toBe(Action.CREATE);
    expect(conditions[3].base.action).toBe(Action.CREATE);
    expect(conditions[4].base.action).toBe(Action.CREATE);

    // First two conditions should share an orGroupId
    expect(conditions[0].base.orGroupId).toBe(conditions[1].base.orGroupId);
    // Third condition should have no orGroupId
    expect(conditions[2].base.orGroupId).toBe('');
    // Fourth condition should have a different orGroupId
    expect(conditions[3].base.orGroupId).not.toBe(conditions[0].base.orGroupId);
    expect(conditions[3].base.orGroupId).not.toBe('');
    expect(conditions[3].base.orGroupId).not.toBe(conditions[2].base.orGroupId);
    // Fifth condition should have a different orGroupId
    expect(conditions[4].base.orGroupId).not.toBe(conditions[0].base.orGroupId);
    expect(conditions[4].base.orGroupId).not.toBe('');
    expect(conditions[4].base.orGroupId).not.toBe(conditions[2].base.orGroupId);
  });

  // Render function edge cases
  it('should handle rendering of empty OR conditions', () => {
    const orCondition = new OrCondition([]);
    expect(render(orCondition)).toBe('OR()');
  });

  it('should handle undefined expression in UserEventCondition', () => {
    const userEvent: UserEvent = { eventKey: 'test:event' };
    const condition = Render(Action.CREATE, userEvent)[0] as UserEventCondition;
    expect(render(condition)).toBe('event: test:event');
  });
});
