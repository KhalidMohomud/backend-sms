# Kobciye School Backend API

Backend foundation built with Go 1.26, Gin, GORM, PostgreSQL, Redis, JWT, Swagger, and Docker. The code follows a handler → service → repository structure.

## Run with Docker

```sh
cp .env.example .env
docker compose up --build
```

The API is available at `http://localhost:8081`, health status at `/health`, and Swagger UI at `/swagger/index.html`. Set `API_HOST_PORT` to use another host port; the container always listens on port 8080 internally.

PostgreSQL and Redis are available to the API on Docker's private network and are not published to host ports, avoiding conflicts with locally installed database services.

## Run locally

Start PostgreSQL and Redis, configure `.env` values in your shell, then run:

```sh
make run
```

Environment variables are read from the process environment; `.env` is consumed automatically by Docker Compose.

## API endpoints

- `GET /health`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`

Protected routes can use `middleware.Authenticate(jwtManager)` and read the authenticated user ID from `middleware.UserIDKey`.

## Development

```sh
make fmt
make test
make swagger
```
# backend-sms
