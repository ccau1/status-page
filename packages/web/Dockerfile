# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Development stage with Vite SSR HMR
# -----------------------------------------------------------------------------
FROM node:24-alpine AS dev

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

EXPOSE 3000

ENV NODE_ENV=development
ENV PORT=3000

CMD ["npm", "run", "dev"]

# -----------------------------------------------------------------------------
# Production build stage
# -----------------------------------------------------------------------------
FROM node:24-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

# Final production runner
FROM node:24-alpine AS prod

WORKDIR /app

COPY package*.json ./
RUN npm install --omit=dev

COPY --from=builder /app/dist ./dist
COPY --from=builder /app/server.js ./server.js

EXPOSE 3000

ENV NODE_ENV=production
ENV PORT=3000

CMD ["node", "server.js"]
