# syntax=docker/dockerfile:1
FROM node:20-alpine AS deps
WORKDIR /src
COPY apps/web/package.json apps/web/package-lock.json* ./
RUN npm ci

FROM node:20-alpine AS build
WORKDIR /src
COPY apps/web/ ./
COPY --from=deps /src/node_modules ./node_modules
RUN npm run build

FROM node:20-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /src/.next ./.next
COPY --from=build /src/public ./public
COPY --from=build /src/package.json ./package.json
COPY --from=build /src/node_modules ./node_modules
EXPOSE 3000
CMD ["npm", "run", "start"]
