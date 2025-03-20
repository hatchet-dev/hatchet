import { Condition, Action } from './base';

export interface UserEvent {
  eventKey: string;
  expression?: string;
}

export class UserEventCondition extends Condition {
  eventKey: string;
  expression: string;

  constructor(eventKey: string, expression: string, action?: Action) {
    super({
      readableDataKey: '',
      action,
      orGroupId: '',
      expression: '',
    });
    this.eventKey = eventKey;
    this.expression = expression;
  }
}
