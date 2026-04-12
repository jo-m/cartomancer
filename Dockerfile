# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
COPY internal/pkg/api/openapi.yaml /app/internal/pkg/api/openapi.yaml
RUN npm run build

# Stage 2: Build backend
FROM golang:1.26.2-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/static ./static
RUN go generate ./...
RUN go build -trimpath -ldflags="-s -w -linkmode external -extldflags '-static'" -o /cartomancer .
RUN mkdir /data

# Stage 3: Final minimal image
FROM gcr.io/distroless/static-debian13:latest
COPY --from=backend /cartomancer /cartomancer
COPY --from=backend --chown=root:root /data /data
WORKDIR /
ENTRYPOINT ["/cartomancer"]
CMD ["serve"]

ENV LISTEN_ADDR=0.0.0.0:8080
