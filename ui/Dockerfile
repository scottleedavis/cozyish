FROM node:lts-alpine

WORKDIR /app

COPY package*.json ./

COPY . .

RUN npm install

EXPOSE 7777

CMD npm run serve
