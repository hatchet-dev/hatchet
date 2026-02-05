import { Logger, LogLevel } from '@hatchet-dev/typescript-sdk/util/logger';
import pino from 'pino';
import Hatchet from '@hatchet-dev/typescript-sdk/sdk';
import { JsonObject } from '@hatchet-dev/typescript-sdk/v1';

// > Create Pino logger
const logger = pino();

class PinoLogger implements Logger {
  logLevel: LogLevel;
  context: string;

  constructor(context: string, logLevel: LogLevel = 'DEBUG') {
    this.logLevel = logLevel;
    this.context = context;
  }

  debug(message: string, extra?: JsonObject): void {
    logger.debug(extra, message);
  }

  info(message: string, extra?: JsonObject): void {
    logger.info(extra, message);
  }

  green(message: string, extra?: JsonObject): void {
    logger.info(extra, `%c${message}`);
  }

  warn(message: string, error?: Error, extra?: JsonObject): void {
    logger.warn(extra, `${message} ${error}`);
  }

  error(message: string, error?: Error, extra?: JsonObject): void {
    logger.error(extra, `${message} ${error}`);
  }

  // optional util method
  util(key: string, message: string, extra?: JsonObject): void {
    // for example you may want to expose a trace method
    if (key === 'trace') {
      logger.info(extra, 'trace');
    }
  }
}

const hatchet = Hatchet.init({
  log_level: 'DEBUG',
  logger: (ctx, level) => new PinoLogger(ctx, level),
});


// > Use the logger

const workflow = hatchet.task({
  name: 'byo-logger-example',
  fn: async (ctx) => {
    for (let i = 0; i < 5; i++) {
      logger.info(`log message ${i}`);
    }

    return { step1: 'completed step run' };
  },
});


async function main() {
  const worker = await hatchet.worker('byo-logger-worker', {
    workflows: [workflow],
  });
  worker.start();
}

main();

