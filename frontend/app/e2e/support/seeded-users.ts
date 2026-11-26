import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export type SeededUserKey = 'owner' | 'admin' | 'member';

type SeededUser = {
  name: string;
  email: string;
  password: string;
  role: string;
  userId: string;
};

const SEEDED_USERS_PATH = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  'seeded-users.generated.json',
);

let cachedUsers: Record<SeededUserKey, SeededUser> | undefined;

// Read on first use rather than imported, so isolated tests (and anything else
// that never logs in) can run without a seeded database.
export function seededUser(key: SeededUserKey): SeededUser {
  if (!cachedUsers) {
    if (!fs.existsSync(SEEDED_USERS_PATH)) {
      throw new Error(
        [
          `Missing ${SEEDED_USERS_PATH}.`,
          'Seed the database before running @seeded tests:',
          '  task seed-e2e',
          'or',
          '  SEED_DEVELOPMENT=true bash ./hack/dev/run-go-with-env.sh run ./cmd/hatchet-admin seed-e2e',
        ].join('\n'),
      );
    }

    cachedUsers = JSON.parse(fs.readFileSync(SEEDED_USERS_PATH, 'utf8'));
  }

  return cachedUsers![key];
}
