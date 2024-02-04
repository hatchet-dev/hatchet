import Hatchet from '../src/sdk';
import { HatchetClient } from '../src/clients/hatchet-client/hatchet-client';
import { StepRunEvent } from '../src/clients/listener/listener-client';

const hatchet: HatchetClient = Hatchet.init();

async function main() {
  const workflowRunId = await hatchet.admin.run_workflow('example', {});

  hatchet.listener.on(workflowRunId, async (event: StepRunEvent) => {
    console.log('Received event', event);
  });
}

main();
