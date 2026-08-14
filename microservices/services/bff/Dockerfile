FROM node:24-alpine AS build
WORKDIR /app
RUN corepack enable
COPY services/bff/package.json services/bff/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-workspace

COPY services/bff/tsconfig.json ./
COPY services/bff/src ./src
# 型を落として JS にする。実行時に TypeScript のツールを持たせない
RUN pnpm build

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production
RUN corepack enable
COPY services/bff/package.json services/bff/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --prod --ignore-workspace
COPY --from=build /app/dist ./dist
# 非 root で動かす（node イメージには node ユーザーが用意されている）
USER node
EXPOSE 3000
CMD ["node", "dist/main.js"]
