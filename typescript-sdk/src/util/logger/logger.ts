export type LogLevel = 'OFF' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

// eslint-disable-next-line no-shadow
enum LogLevelEnum {
  OFF = -1,
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
}

export class Logger {
  private logLevel: LogLevel;
  private context: string;

  constructor(context: string, logLevel: LogLevel = 'INFO') {
    this.logLevel = logLevel;
    this.context = context;
  }

  private log(level: LogLevel, message: string): void {
    if (LogLevelEnum[level] >= LogLevelEnum[this.logLevel]) {
      console.log(`🪓 [${level}/${this.context}] ${message}`);
    }
  }

  debug(message: string): void {
    this.log('DEBUG', message);
  }

  info(message: string): void {
    this.log('INFO', message);
  }

  warn(message: string): void {
    this.log('WARN', message);
  }

  error(message: string): void {
    this.log('ERROR', message);
  }
}

export default Logger;
