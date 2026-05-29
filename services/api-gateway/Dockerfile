FROM node:22-alpine

WORKDIR /app

COPY package*.json ./

RUN npm ci --omit=dev --ignore-scripts

COPY . .

EXPOSE 4000

CMD ["npm", "start"]