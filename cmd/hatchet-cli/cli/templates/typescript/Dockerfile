FROM node:22

RUN apt-get update -y && apt-get install -y openssl

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY src ./src
COPY tsconfig.json ./
RUN npm run build

CMD ["npm", "run", "start"]