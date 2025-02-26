export abstract class Logger {
  abstract debug(message: string): void;
  abstract info(message: string): void;
  abstract green(message: string): void;
  abstract warn(message: string, error?: Error): void;
  abstract error(message: string, error?: Error): void;
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
