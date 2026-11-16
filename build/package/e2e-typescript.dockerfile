# Base Node environment
# ---------------------
FROM node:24-alpine AS deployment

WORKDIR /hatchet/sdks/typescript

RUN corepack enable && corepack prepare pnpm@10.16.1 --activate

COPY sdks/typescript/ .

RUN pnpm install --frozen-lockfile

CMD ["pnpm", "run", "test:e2e"]
