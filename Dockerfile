# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
COPY internal/pkg/rest/openapi.yaml /app/internal/pkg/rest/openapi.yaml
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/static ./static
RUN go generate ./...
RUN go build -trimpath -ldflags="-s -w -linkmode external -extldflags '-static'" -o /detour .
RUN mkdir /data

# Stage 3: Final minimal image
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=backend /detour /detour
COPY --from=backend --chown=nonroot:nonroot /data /home/nonroot/data
WORKDIR /home/nonroot
ENTRYPOINT ["/detour"]

ENV LOG_PRETTY=true
ENV LISTEN_ADDR=0.0.0.0:8080
ENV APP_INIT_ADMIN_EMAIL=admin@example.com
