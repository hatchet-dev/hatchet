import { Logger, LogLevel, LogLevelEnum } from '@hatchet/util/logger/logger';

/**
 * The logger the core client uses when none is configured: level-filtered `console` output
 * with the context in brackets. It reads nothing from the process, so it runs anywhere the
 * client does.
 */
export class ConsoleLogger implements Logger {
  /** The lowest level written; `OFF` sets it above every level, so nothing is. */
  private readonly threshold: number;

  constructor(
    private readonly context: string,
    logLevel: LogLevel = 'INFO'
  ) {
    this.threshold = logLevel === 'OFF' ? Number.POSITIVE_INFINITY : LogLevelEnum[logLevel];
  }

  private log(level: LogLevel, message: string, error?: Error) {
    if (LogLevelEnum[level] < this.threshold) return;
    const write =
      level === 'ERROR'
        ? console.error
        : level === 'WARN'
          ? console.warn
          : level === 'DEBUG'
            ? console.debug
            : console.info;
    write(`[${level}/${this.context}] ${error ? `${message} ${error}` : message}`);
  }

  debug(message: string) {
    this.log('DEBUG', message);
  }

  info(message: string) {
    this.log('INFO', message);
  }

  green(message: string) {
    this.log('INFO', message);
  }

  warn(message: string, error?: Error) {
    this.log('WARN', message, error);
  }

  error(message: string, error?: Error) {
    this.log('ERROR', message, error);
  }
}

export const consoleLogger = (context: string, logLevel?: LogLevel): Logger =>
  new ConsoleLogger(context, logLevel);
