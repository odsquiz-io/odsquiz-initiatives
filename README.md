# ODS Quiz Initiatives API

Initiatives API for ODS Quiz. It uses Go, Fiber, GORM, and PostgreSQL.

## Run locally

Prerequisites:

- Go version declared in `go.mod`.
- PostgreSQL listening on `127.0.0.1:5432`.
- A local database and role that match your `.env.local` credentials.

Check that PostgreSQL is available:

```bash
pg_isready -h 127.0.0.1 -p 5432
```

Create the local configuration once:

```bash
cp .env.example .env.local
```

Set the values in `.env.local` without committing it:

```dotenv
PORT=8081
JWTSecret=local-development-secret
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=your_local_user
DB_PASSWORD=your_local_password
DB_NAME=odsquiz
DB_SSLMODE=disable
```

Run the API:

```bash
go run ./cmd/api
```

At startup, the API checks the database connection, retries transient failures
with exponential backoff, and runs its GORM migration. Verify it in another
terminal:

```bash
curl --fail-with-body http://127.0.0.1:8081/health
```

Expected response:

```text
ok
```

## Validation

```bash
go test ./...
go vet ./...
docker build -t odsquiz-initiatives .
```

## API paths

- `POST /api/initiatives`
- `GET /api/initiatives`
- `GET /api/initiatives/mine`
- `GET /api/initiatives/:id`
- `PATCH /api/initiatives/:id`
- `DELETE /api/initiatives/:id`
- `GET /health`

Initiative CRUD routes require a bearer token from the auth API. The token's
`sub` claim becomes the owner of created initiatives and selects the ones
returned by `GET /api/initiatives/mine`, newest first. The public load balancer
exposes this service through `/api/initiatives/*`.

`POST` and `PATCH` accept only `name`, `address`, `cep`, `city`, `state`,
`email_owner`, `acting_area`, `impact`, `type`, `main_ods`, `lat`, and `lon`;
other body fields such as `owner`, `points`, or the suspension fields are
ignored. `PATCH` changes only the fields sent with non-zero values and returns
the full initiative.

Moderation happens in the admin service; this API enforces it:

- A user whose `users.suspended_at` is set gets
  `403 {"code":"account_suspended"}` on `POST`, `PATCH`, and `DELETE`.
- Only the owner can `PATCH` or `DELETE` an initiative; anyone else gets
  `403 {"code":"forbidden"}`.
- Suspended initiatives (`suspended_at` set) are left out of
  `GET /api/initiatives`, and `GET /api/initiatives/:id` answers
  `404 {"code":"initiative_not_found"}` for them unless the requester owns
  them. `GET /api/initiatives/mine` keeps them, with `suspended_at` and
  `suspension_reason`.

The older aliases (`/createOne`, `/getAllOnes`, `/getOneById/:id`,
`/updateOne/:id`, `/deleteOne/:id`) share these handlers and rules.

## Configuration convention

Use `DB_SSLMODE` in all new configuration. The application temporarily accepts
the older `DBSSLMode` spelling to avoid breaking existing developer machines.
