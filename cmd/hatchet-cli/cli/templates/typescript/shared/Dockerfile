FROM node:22

RUN apt-get update -y && apt-get install -y openssl

WORKDIR /app

{{- if eq .PackageManager "npm"}}
COPY package*.json ./
RUN npm install

COPY src ./src
COPY tsconfig.json ./
RUN npm run build

CMD ["npm", "run", "start"]
{{- else if eq .PackageManager "pnpm"}}
RUN npm install -g pnpm
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY src ./src
COPY tsconfig.json ./
RUN pnpm run build

CMD ["pnpm", "start"]
{{- else if eq .PackageManager "yarn"}}
COPY package.json ./
RUN yarn install

COPY src ./src
COPY tsconfig.json ./
RUN yarn build

CMD ["yarn", "start"]
{{- else if eq .PackageManager "bun"}}
COPY --from=oven/bun:latest /usr/local/bin/bun /usr/local/bin/bun
COPY package.json ./
RUN bun install

COPY src ./src
COPY tsconfig.json ./
RUN bun run build

CMD ["bun", "start"]
{{- end}}
