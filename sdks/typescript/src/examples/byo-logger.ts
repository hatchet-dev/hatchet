import { Logger, LogLevel } from '@hatchet/util/logger';
// eslint-disable-next-line import/no-extraneous-dependencies
import pino from 'pino';
import Hatchet from '@hatchet/sdk';
import { JsonObject } from '@hatchet/v1';

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
    logger.debug(message, extra);
  }

  info(message: string, extra?: JsonObject): void {
    logger.info(message, extra);
  }

  green(message: string, extra?: JsonObject): void {
    logger.info(`%c${message}`, extra);
  }

  warn(message: string, error?: Error, extra?: JsonObject): void {
    logger.warn(`${message} ${error}`, extra);
  }

  error(message: string, error?: Error, extra?: JsonObject): void {
    logger.error(`${message} ${error}`, extra);
  }

  // optional util method
  util(key: string, message: string, extra?: JsonObject): void {
    // for example you may want to expose a trace method
    if (key === 'trace') {
      logger.info('trace', extra);
    }
  }
}

const hatchet = Hatchet.init({
  log_level: 'DEBUG',
  logger: (ctx, level) => new PinoLogger(ctx, level),
});

// !!

// > Use the logger

const workflow = hatchet.task({
  name: 'byo-logger-example',
  fn: async (ctx) => {
    // eslint-disable-next-line no-plusplus
    for (let i = 0; i < 5; i++) {
      logger.info(`log message ${i}`);
    }

    return { step1: 'completed step run' };
  },
});

// !!

async function main() {
  const worker = await hatchet.worker('byo-logger-worker', {
    workflows: [workflow],
  });
  worker.start();
}

main();
