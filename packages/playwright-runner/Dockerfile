# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Development & Production image with official Playwright dependencies
# -----------------------------------------------------------------------------
FROM mcr.microsoft.com/playwright:v1.51.1-noble AS base

WORKDIR /app

ENV PORT=8088
ENV NODE_ENV=production

# Copy package manifests
COPY packages/playwright-runner/package*.json ./

# Install production dependencies
RUN npm ci --only=production

# Development stage
FROM base AS dev
ENV NODE_ENV=development
RUN npm install
COPY packages/playwright-runner ./
EXPOSE 8088
CMD ["npm", "run", "dev"]

# Production build stage
FROM base AS builder
RUN npm install
COPY packages/playwright-runner ./
RUN npm run build

# Final production stage
FROM base AS prod
COPY --from=builder /app/dist ./dist
EXPOSE 8088
CMD ["node", "dist/src/index.js"]
