FROM node:25 AS assets
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts
RUN npm run vendor

# -----------------------------------------------------

FROM golang:1.27.2 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY www www
COPY --from=assets /build/www/vendor www/vendor

# compile with static linking
RUN CGO_ENABLED=0 go build -o /casino .

# -----------------------------------------------------

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /casino /casino

# Declared explicitly to satisfy static scanners; the :nonroot tag already
# defaults to this same user (UID 65532).
USER nonroot:nonroot

EXPOSE 8080
ENTRYPOINT [ "/casino" ]
