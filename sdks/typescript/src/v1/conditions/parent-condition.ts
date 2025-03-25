import { Condition, Action } from './base';
import { CreateWorkflowTaskOpts } from '../task';

export interface Parent {
  parent: CreateWorkflowTaskOpts<any, any>;
  expression?: string;
}

export class ParentCondition extends Condition {
  parent: CreateWorkflowTaskOpts<any, any>;

  constructor(parent: CreateWorkflowTaskOpts<any, any>, expression?: string, action?: Action) {
    super({
      readableDataKey: '',
      action,
      orGroupId: '',
      expression: expression || '',
    });
    this.parent = parent;
  }
}
