# syntax=docker/dockerfile:1

# 1. Build the front-end.
FROM node:26-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2. Build a static Go binary with the front-end embedded.
FROM golang:1.27-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /painel ./cmd/server

# 3. Run it on a minimal image: no shell, CA certificates included, non-root.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=server /painel /painel
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/painel"]
