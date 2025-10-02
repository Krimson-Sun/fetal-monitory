# Используем официальный образ Node.js 20 для сборки (без указания slim/alpine)
FROM node:20 AS builder
WORKDIR /app

# Копируем только package.json и package-lock.json
COPY package*.json ./

# Устанавливаем зависимости с очисткой кэша
RUN npm ci --no-fund --no-audit

# Копируем остальные файлы проекта
COPY . .

# Удаляем проблемные нативные модули Rollup и переустанавливаем Rollup
RUN rm -rf node_modules/@rollup && \
    npm install rollup@latest --no-fund --no-audit && \
    npm rebuild --no-fund --no-audit

# Собираем проект
RUN npm run build

# Оптимизированный образ Nginx для продакшена
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80