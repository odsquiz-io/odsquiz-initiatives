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

Initiatives:

- `POST /api/initiatives`
- `GET /api/initiatives`
- `GET /api/initiatives/mine`
- `GET /api/initiatives/:id`
- `PATCH /api/initiatives/:id`
- `DELETE /api/initiatives/:id`

Logo, members and invitations:

- `GET /api/initiatives/:id/logo` (public, no token)
- `PUT /api/initiatives/:id/logo` (multipart field `logo`)
- `DELETE /api/initiatives/:id/logo`
- `GET /api/initiatives/:id/members`
- `POST /api/initiatives/:id/members`
- `DELETE /api/initiatives/:id/members/:memberId`
- `GET /api/initiatives/invitations`
- `POST /api/initiatives/invitations/:memberId/accept`
- `POST /api/initiatives/invitations/:memberId/decline`
- `GET /api/initiatives/memberships`

Posts and comments:

- `GET /api/initiatives/feed`
- `GET /api/initiatives/:id/posts`
- `POST /api/initiatives/:id/posts`
- `PATCH /api/initiatives/posts/:postId`
- `DELETE /api/initiatives/posts/:postId`
- `GET /api/initiatives/posts/:postId/comments`
- `POST /api/initiatives/posts/:postId/comments`
- `PATCH /api/initiatives/comments/:commentId`
- `DELETE /api/initiatives/comments/:commentId`

- `GET /health`

Every route except the logo `GET` and `/health` requires a bearer token from
the auth API. The token's `sub` claim becomes the owner of created initiatives
and selects the ones returned by `GET /api/initiatives/mine`, newest first. The
public load balancer exposes this service through `/api/initiatives/*`. Literal
segments (`mine`, `feed`, `invitations`, `memberships`, `posts`, `comments`) are
registered before `/api/initiatives/:id`.

`POST` and `PATCH` accept only `name`, `address`, `cep`, `city`, `state`,
`email_owner`, `acting_area`, `impact`, `type`, `main_ods`, `lat`, `lon`, and
`description` (up to 2000 characters); other body fields such as `owner`,
`points`, or the suspension fields are ignored. `PATCH` changes exactly the
fields present in the body, zero values included, and returns the full
initiative. Responses expose `description` and `logo_url`; `GET /:id` also
returns `viewer_role` (`owner`, `member` or `null`) and `members_count`.

Moderation happens in the admin service; this API enforces it:

- A user whose `users.suspended_at` is set gets
  `403 {"code":"account_suspended"}` on every write.
- Only the owner can `PATCH` or `DELETE` an initiative, manage its logo and
  invite or remove members; anyone else gets `403 {"code":"forbidden"}`.
- Suspended initiatives (`suspended_at` set) are left out of
  `GET /api/initiatives`, and `GET /api/initiatives/:id` answers
  `404 {"code":"initiative_not_found"}` for them unless the requester owns
  them. `GET /api/initiatives/mine` keeps them, with `suspended_at` and
  `suspension_reason`.
- Posts and comments are hidden when they are suspended, when their author is
  suspended, or when their initiative is suspended.

Members join by invitation: the owner invites a registered user by e-mail and
the invitee accepts or declines; a member can leave at any time. The owner and
active members publish posts; any active user comments. Authors edit their own
posts and comments; authors and the initiative owner delete them. Deleting an
initiative also deletes its members, posts, comments and logo.

The older aliases (`/createOne`, `/getAllOnes`, `/getOneById/:id`,
`/updateOne/:id`, `/deleteOne/:id`) share these handlers and rules.

## Logo storage

Logos live in Google Cloud Storage, in the bucket named by `STORAGE_BUCKET`.
On Cloud Run the client uses the service account through Application Default
Credentials; locally it talks to the emulator set by `STORAGE_EMULATOR_HOST`.
Without `STORAGE_BUCKET` the service still starts and the logo routes answer
`503 {"code":"storage_unavailable"}`.

## Configuration convention

Use `DB_SSLMODE` in all new configuration. The application temporarily accepts
the older `DBSSLMode` spelling to avoid breaking existing developer machines.
