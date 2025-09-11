import {
  TaskConditions,
  ParentOverrideMatchCondition,
  SleepMatchCondition,
  UserEventMatchCondition,
  BaseMatchCondition,
} from '@hatchet/protoc/v1/shared/condition';
import { Render, SleepCondition, UserEventCondition, generateGroupId } from '.';
import { CreateWorkflowTaskOpts } from '../task';
import { Action, BaseCondition, Condition } from './base';
import { ParentCondition } from './parent-condition';

export function taskConditionsToPb(
  task: Omit<CreateWorkflowTaskOpts<any, any>, 'fn'>
): TaskConditions {
  const waitForConditions = Render(Action.QUEUE, task.waitFor);
  const cancelIfConditions = Render(Action.CANCEL, task.cancelIf);
  const skipIfConditions = Render(Action.SKIP, task.skipIf);
  const mergedConditions = [...waitForConditions, ...cancelIfConditions, ...skipIfConditions];

  return conditionsToPb(mergedConditions);
}

export function conditionsToPb(conditions: Condition[]): TaskConditions {
  const parentOverrideConditions: ParentOverrideMatchCondition[] = [];
  const sleepConditions: SleepMatchCondition[] = [];
  const userEventConditions: UserEventMatchCondition[] = [];

  conditions.forEach((condition) => {
    if (condition instanceof SleepCondition) {
      sleepConditions.push({
        base: baseToPb(condition.base),
        sleepFor: condition.sleepFor,
      });
    } else if (condition instanceof UserEventCondition) {
      userEventConditions.push({
        base: {
          ...baseToPb(condition.base),
          expression: condition.expression || '',
        },
        userEventKey: condition.eventKey,
      });
    } else if (condition instanceof ParentCondition) {
      parentOverrideConditions.push({
        base: baseToPb(condition.base),
        parentReadableId: condition.parent.name,
      });
    }
  });

  return {
    parentOverrideConditions,
    sleepConditions,
    userEventConditions,
  };
}

function baseToPb(base: BaseCondition): BaseMatchCondition {
  return {
    readableDataKey: base.readableDataKey || '',
    action: base.action!,
    orGroupId: base.orGroupId || generateGroupId(),
    expression: base.expression || '',
  };
}
