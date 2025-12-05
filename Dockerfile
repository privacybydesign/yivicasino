FROM node:18 AS node

WORKDIR /build

COPY www/package*.json .

RUN npm i

# ---

FROM composer:latest AS composer
FROM dunglas/frankenphp:1.10.1-php8.3-bookworm

ENV SERVER_NAME=:8080
ENV DEBIAN_FRONTEND=noninteractive

# Enable PHP production settings
RUN mv "$PHP_INI_DIR/php.ini-production" "$PHP_INI_DIR/php.ini" \
    && apt update && apt upgrade -y \
    && install-php-extensions zip

COPY ./data /app/data
COPY ./www /app/public

COPY --from=composer /usr/bin/composer /usr/local/bin/composer
COPY --from=node /build/node_modules /app/public/node_modules

RUN cd /app/public \
    && composer install --no-interaction --optimize-autoloader --no-dev

EXPOSE 8080

