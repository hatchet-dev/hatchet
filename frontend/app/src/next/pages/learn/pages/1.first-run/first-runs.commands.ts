import { Commands } from '../../components/lesson-plan';

export interface CommandConfig {
  install: string;
  startWorker: string;
  runTask: string;
}

export const commands: Commands<CommandConfig> = {
  pnpm: {
    install: 'pnpm install',
    startWorker: 'pnpm run dev',
    runTask: 'pnpm run task',
  },
  npm: {
    install: 'npm install',
    startWorker: 'npm run dev',
    runTask: 'npm run task',
  },
  yarn: {
    install: 'yarn install',
    startWorker: 'yarn dev',
    runTask: 'yarn run task',
  },
  poetry: {
    install: 'poetry install',
    startWorker: 'poetry run python main.py',
    runTask: 'poetry run python main.py',
  },
  uv: {
    install: 'pip install -r requirements.txt',
    startWorker: 'python main.py',
    runTask: 'python main.py',
  },
  pip: {
    install: 'pip install -r requirements.txt',
    startWorker: 'python main.py',
    runTask: 'python main.py',
  },
  pipenv: {
    install: 'pipenv install',
    startWorker: 'pipenv run python main.py',
    runTask: 'pipenv run python main.py',
  },
  go: {
    install: 'go mod tidy',
    startWorker: 'go run main.go',
    runTask: 'go run main.go',
  },
};
