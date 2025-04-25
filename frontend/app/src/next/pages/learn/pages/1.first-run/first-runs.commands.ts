import { Commands, SupportedLanguage } from '@/next/learn/components';

interface SharedCommandConfig {
  repo: string;
  clone: string;
}

export interface CommandConfig extends SharedCommandConfig {
  install: string;
  startWorker: string;
  runTask: string;
}

const shared: Record<SupportedLanguage, SharedCommandConfig> = {
  typescript: {
    repo: 'hatchet-ts-quickstart',
    clone: `git clone https://github.com/hatchet-dev/hatchet-ts-quickstart &&\ncd hatchet-ts-quickstart`,
  },
  python: {
    repo: 'hatchet-python-quickstart',
    clone: `git clone https://github.com/hatchet-dev/hatchet-python-quickstart &&\ncd hatchet-python-quickstart`,
  },
  go: {
    repo: 'hatchet-go-quickstart',
    clone: `git clone https://github.com/hatchet-dev/hatchet-go-quickstart &&\ncd hatchet-go-quickstart`,
  },
};

export const commands: Commands<CommandConfig> = {
  pnpm: {
    ...shared.typescript,
    install: 'pnpm install',
    startWorker: 'pnpm run start',
    runTask: 'pnpm run run:simple',
  },
  npm: {
    ...shared.typescript,
    install: 'npm install',
    startWorker: 'npm run start',
    runTask: 'npm run run:simple',
  },
  yarn: {
    ...shared.typescript,
    install: 'yarn install',
    startWorker: 'yarn run start',
    runTask: 'yarn run run:simple',
  },
  poetry: {
    ...shared.python,
    install: 'poetry install',
    startWorker: 'poetry run python main.py',
    runTask: 'poetry run python main.py',
  },
  uv: {
    ...shared.python,
    install: 'pip install -r requirements.txt',
    startWorker: 'python main.py',
    runTask: 'python main.py',
  },
  pip: {
    ...shared.python,
    install: 'pip install -r requirements.txt',
    startWorker: 'python main.py',
    runTask: 'python main.py',
  },
  pipenv: {
    ...shared.python,
    install: 'pipenv install',
    startWorker: 'pipenv run python main.py',
    runTask: 'pipenv run python main.py',
  },
  go: {
    ...shared.go,
    install: 'go mod tidy',
    startWorker: 'go run main.go',
    runTask: 'go run main.go',
  },
};
