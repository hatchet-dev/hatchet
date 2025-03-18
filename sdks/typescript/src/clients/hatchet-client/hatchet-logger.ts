import { Logger, LogLevel, LogLevelEnum } from '@util/logger';

export const DEFAULT_LOGGER = (context: string, logLevel?: LogLevel) =>
  new HatchetLogger(context, logLevel);

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
      // eslint-disable-next-line no-console
      console.log(
        `🪓 ${process.pid} | ${time} ${color && `\x1b[${color || ''}m`} [${level}/${this.context}] ${message}\x1b[0m`
      );
    }
  }

  debug(message: string): void {
    this.log('DEBUG', message, '35');
  }

  info(message: string): void {
    this.log('INFO', message);
  }

  green(message: string): void {
    this.log('INFO', message, '32');
  }

  warn(message: string, error?: Error): void {
    this.log('WARN', `${message} ${error}`, '93');
  }

  error(message: string, error?: Error): void {
    this.log('ERROR', `${message} ${error}`, '91');
  }
}
