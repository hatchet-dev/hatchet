export abstract class Logger {
  abstract debug(message: string, extra?: any): void;
  abstract info(message: string, extra?: any): void;
  abstract green(message: string, extra?: any): void;
  abstract warn(message: string, error?: Error, extra?: any): void;
  abstract error(message: string, error?: Error, extra?: any): void;
}

export type LogLevel = 'OFF' | 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

// eslint-disable-next-line no-shadow
export enum LogLevelEnum {
  OFF = -1,
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
}
