import { hatchet } from '../hatchet-client';

export type Input = {
  Message: string;
};

type OnCronOutput = {
  job: {
    TransformedMessage: string;
  };
};

// > Run Workflow on Cron
export const onCron = hatchet.workflow<Input, OnCronOutput>({
  name: 'on-cron-workflow',
  on: {
    // 👀 add a cron expression to run the workflow every 15 minutes
    cron: '*/15 * * * *',
  },
});

onCron.task({
  name: 'job',
  fn: (input) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
});
