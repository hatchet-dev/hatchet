import { randomUUID } from 'crypto';

import { hatchet } from '../hatchet-client';

type Input = {
  prompt: string;
  runId?: string;
};

type Output = {
  'governed-model-call': {
    runId: string;
    response: any;
  };
};

export const tuningEnginesWorkflow = hatchet.workflow<Input, Output>({
  name: 'tuning-engines-governed-ai',
});

tuningEnginesWorkflow.task({
  name: 'governed-model-call',
  fn: async (input) => {
    const runId = input.runId || randomUUID();
    const response = await fetch('https://api.tuningengines.com/v1/chat/completions', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${process.env.TE_INFERENCE_KEY}`,
        'Content-Type': 'application/json',
        'X-TE-Run-ID': runId,
      },
      body: JSON.stringify({
        model: process.env.TE_MODEL || 'auto',
        messages: [{ role: 'user', content: input.prompt }],
        metadata: {
          run_id: runId,
          request_id: randomUUID(),
          runtime: 'hatchet',
          event_type: 'model.call',
        },
      }),
    });

    if (!response.ok) {
      throw new Error(`Tuning Engines request failed: ${response.status}`);
    }

    return {
      runId,
      response: await response.json(),
    };
  },
});
