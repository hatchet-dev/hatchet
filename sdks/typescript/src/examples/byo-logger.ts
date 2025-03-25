import { Logger, LogLevel } from '@hatchet/util/logger';
// eslint-disable-next-line import/no-extraneous-dependencies
import pino from 'pino';
import Hatchet from '../sdk';
import { Workflow } from '../workflow';

// ❓ Create Pino logger
const logger = pino();

class PinoLogger implements Logger {
  logLevel: LogLevel;
  context: string;

  constructor(context: string, logLevel: LogLevel = 'DEBUG') {
    this.logLevel = logLevel;
    this.context = context;
  }

  debug(message: string): void {
    logger.debug(message);
  }

  info(message: string): void {
    logger.info(message);
  }

  green(message: string): void {
    logger.info(`%c${message}`);
  }

  warn(message: string, error?: Error): void {
    logger.warn(`${message} ${error}`);
  }

  error(message: string, error?: Error): void {
    logger.error(`${message} ${error}`);
  }
}

const hatchet = Hatchet.init({
  log_level: 'DEBUG',
  logger: (ctx, level) => new PinoLogger(ctx, level),
});

// !!

// ❓ Use the logger

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

const workflow: Workflow = {
  id: 'byo-logger-example',
  description: 'An example showing how to pass a custom logger to Hatchet',
  on: {
    event: 'byo-logger:spawn',
  },
  steps: [
    {
      name: 'logger-step1',
      run: async (ctx) => {
        // eslint-disable-next-line no-plusplus
        for (let i = 0; i < 5; i++) {
          logger.info(`log message ${i}`);
          await sleep(500);
        }

        return { step1: 'completed step run' };
      },
    },
  ],
};

// !!

async function main() {
  const worker = await hatchet.worker('byo-logger-worker', 1);
  await worker.registerWorkflow(workflow);
  worker.start();
}

main();
