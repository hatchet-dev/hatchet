import Hatchet from '../sdk';
import { Workflow } from '../workflow';

const hatchet = Hatchet.init();

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// ❓ SlotRelease

const workflow: Workflow = {
    steps: [
        {
        run: async (ctx) => {
            console.log("RESOURCE INTENSIVE PROCESS...");
            await sleep(5000);
            // Release the slot after the resource-intensive process, so that other steps can run
            await ctx.releaseSlot();
            console.log("NON RESOURCE INTENSIVE PROCESS...");
            return { step1: "step1 results!" };
        },
        },
    ],
    };
// ‼️