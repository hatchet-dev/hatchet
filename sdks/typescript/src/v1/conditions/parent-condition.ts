import { Condition, Action } from './base';
import { CreateTaskOpts } from '../task';

export interface Parent {
  parent: CreateTaskOpts<any, any>;
  expression?: string;
}

export class ParentCondition extends Condition {
  parent: CreateTaskOpts<any, any>;

  constructor(parent: CreateTaskOpts<any, any>, expression?: string, action?: Action) {
    super({
      readableDataKey: '',
      action,
      orGroupId: '',
      expression: expression || '',
    });
    this.parent = parent;
  }
}
