import {
  TaskConditions,
  ParentOverrideMatchCondition,
  SleepMatchCondition,
  UserEventMatchCondition,
  BaseMatchCondition,
} from '@hatchet/protoc/v1/shared/condition';
import {
  Render,
  Action,
  SleepCondition,
  UserEventCondition,
  BaseCondition,
  generateGroupId,
} from '.';
import { CreateTaskOpts } from '../task';

export function taskConditionsToPb(task: CreateTaskOpts<any, any>): TaskConditions {
  const queueIfConditions = Render(Action.QUEUE, task.queueIf);
  const cancelIfConditions = Render(Action.CANCEL, task.cancelIf);
  const skipIfConditions = Render(Action.SKIP, task.skipIf);
  const mergedConditions = [...queueIfConditions, ...cancelIfConditions, ...skipIfConditions];

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
          eventKey: condition.eventKey,
          expression: condition.expression || '',
        },
        userEventKey: condition.eventKey,
      });
    }
  });

  const res = {
    parentOverrideConditions,
    sleepConditions,
    userEventConditions,
  };

  console.log(JSON.stringify(res, null, 2));

  return res;
}

function baseToPb(base: BaseCondition): BaseMatchCondition {
  return {
    eventKey: base.eventKey || '',
    readableDataKey: base.readableDataKey || '',
    action: base.action!,
    orGroupId: base.orGroupId || generateGroupId(),
    expression: base.expression || '',
  };
}
