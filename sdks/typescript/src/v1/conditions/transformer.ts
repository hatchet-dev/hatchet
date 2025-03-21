import {
  TaskConditions,
  ParentOverrideMatchCondition,
  SleepMatchCondition,
  UserEventMatchCondition,
  BaseMatchCondition,
} from '@hatchet/protoc/v1/shared/condition';
import { Render, SleepCondition, UserEventCondition, generateGroupId } from '.';
import { CreateTaskOpts } from '../task';
import { Action, BaseCondition } from './base';
import { ParentCondition } from './parent-condition';

export function taskConditionsToPb(task: CreateTaskOpts<any, any>): TaskConditions {
  const waitForConditions = Render(Action.QUEUE, task.waitFor);
  const cancelIfConditions = Render(Action.CANCEL, task.cancelIf);
  const skipIfConditions = Render(Action.SKIP, task.skipIf);
  const mergedConditions = [...waitForConditions, ...cancelIfConditions, ...skipIfConditions];

  const parentOverrideConditions: ParentOverrideMatchCondition[] = [];
  const sleepConditions: SleepMatchCondition[] = [];
  const userEventConditions: UserEventMatchCondition[] = [];

  mergedConditions.forEach((condition) => {
    if (condition instanceof SleepCondition) {
      sleepConditions.push({
        base: baseToPb(condition.base),
        sleepFor: condition.sleepFor.toString(), // TODO consistent duration format
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

  const res = {
    parentOverrideConditions,
    sleepConditions,
    userEventConditions,
  };

  // TODO remove
  console.log(JSON.stringify(res, null, 2));

  return res;
}

function baseToPb(base: BaseCondition): BaseMatchCondition {
  return {
    readableDataKey: base.readableDataKey || '',
    action: base.action!,
    orGroupId: base.orGroupId || generateGroupId(),
    expression: base.expression || '',
  };
}
