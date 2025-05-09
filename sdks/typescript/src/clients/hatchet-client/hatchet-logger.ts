/* eslint-disable no-console */
import { LogExtra, Logger, LogLevel, LogLevelEnum } from '@util/logger';

export const DEFAULT_LOGGER = (context: string, logLevel?: LogLevel) =>
  new HatchetLogger(context, logLevel);

type UtilKeys = 'trace';

export class HatchetLogger implements Logger {
  private logLevel: LogLevel;
  private context: string;

  constructor(context: string, logLevel: LogLevel = 'INFO') {
    this.logLevel = logLevel;
    this.context = context;
  }

  private log(level: LogLevel, message: string, color: string = '37'): void {
    if (LogLevelEnum[level] >= LogLevelEnum[this.logLevel]) {
      const time = new Date().toLocaleString('en-US', {
        month: '2-digit',
        day: '2-digit',
        year: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });

      // eslint-disable-next-line prefer-destructuring
      let print = console.log;

      if (level === 'ERROR') {
        print = console.error;
      }

      if (level === 'WARN') {
        print = console.warn;
      }

      if (level === 'INFO') {
        print = console.info;
      }

      if (level === 'DEBUG') {
        print = console.debug;
      }

      // eslint-disable-next-line no-console
      print(
        `🪓 ${process.pid} | ${time} ${color && `\x1b[${color || ''}m`} [${level}/${this.context}] ${message}\x1b[0m`
      );
    }
  }

  async debug(message: string): Promise<void> {
    await this.log('DEBUG', message, '35');
  }

  async info(message: string): Promise<void> {
    await this.log('INFO', message);
  }

  async green(message: string): Promise<void> {
    await this.log('INFO', message, '32');
  }

  async warn(message: string, error?: Error): Promise<void> {
    await this.log('WARN', `${message} ${error}`, '93');
  }

  async error(message: string, error?: Error): Promise<void> {
    await this.log('ERROR', `${message} ${error}`, '91');
  }

  util(key: UtilKeys, message: string, extra?: LogExtra): void | Promise<void> {
    if (key === 'trace') {
      this.log('INFO', `trace: ${message}`, '35');
    }
  }
}
