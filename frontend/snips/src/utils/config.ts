import { config } from '../../snips.config';
import { Processor } from '../processors/processor.interface';
import { snippetProcessor } from '../processors/snippets/snippet.processor';

export type Config = {
  SOURCE_DIRS: { [key: string]: string };
  OUTPUT_DIR: string;
  PRESERVE_FILES: string[] | RegExp[];
  IGNORE_LIST: string[];
  REPLACEMENTS: Array<{
    from: string;
    to: string;
    fileTypes?: string[];
  }>;
  REMOVAL_PATTERNS: Array<{
    regex: string | RegExp;
    description: string;
  }>;
  PROCESSORS?: Processor[];
};

const DEFAULT_PROCESSORS: Processor[] = [snippetProcessor];

export const getConfig = (overrides: Partial<Config> = {}) => {
  return {
    PROCESSORS: [...DEFAULT_PROCESSORS],
    ...config,
    ...overrides,
  };
};
