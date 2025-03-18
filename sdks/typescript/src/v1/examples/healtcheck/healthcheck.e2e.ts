import sleep from '@hatchet/util/sleep';
import axios from 'axios';
import { hatchet } from '../client';
import { simple } from './workflow';
import { Worker } from '../../..';

describe('Healthcheck', () => {
  const port = 8080;
  let worker: Worker;

  beforeEach(async () => {
    worker = await hatchet.worker('healthcheck-worker', {
      workflows: [simple],
      healthcheck: port,
      slots: 10,
    });
    await worker.start();
    // Allow worker to initialize
    await sleep(1000);
  });

  afterEach(async () => {
    await worker.stop();
    await sleep(2000);
  });

  it('should report correct health status during workflow execution', async () => {
    // Check initial health status
    const { data: initialHealth } = await axios.get(`http://localhost:${port}/health`);
    expect(initialHealth.running).toBe(0);
    // Start workflow execution
    const run = hatchet.run(simple, {
      Message: 'hello',
    });

    // Wait for worker to start running the workflow
    await sleep(500);

    // Check health during workflow execution
    const { data: midExecutionHealth } = await axios.get(`http://localhost:${port}/health`);
    expect(midExecutionHealth.running).toBeDefined();
    expect(midExecutionHealth.running).toBe(1);
    // Wait for workflow to complete
    const res = await run;

    // Check health after workflow completion
    const { data: postExecutionHealth } = await axios.get(`http://localhost:${port}/health`);
    expect(postExecutionHealth.running).toBe(0);

    // Verify workflow result
    expect(res.step2).toBeDefined();
  }, 30000); // Set timeout to 30 seconds
});
