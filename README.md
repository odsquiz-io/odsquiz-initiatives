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

Photos:

- `GET /api/initiatives/:id/photos`
- `POST /api/initiatives/:id/photos` (multipart field `photo`, optional `caption`)
- `PUT /api/initiatives/:id/photos/order`
- `PATCH /api/initiatives/:id/photos/:photoId`
- `DELETE /api/initiatives/:id/photos/:photoId`
- `GET /api/initiatives/:id/photos/:photoId/image` (public, no token)

- `GET /health`

Every route except the logo and photo image `GET`s and `/health` requires a
bearer token from the auth API. The token's `sub` claim becomes the owner of
created initiatives and selects the ones returned by `GET /api/initiatives/mine`,
newest first. The public load balancer exposes this service through
`/api/initiatives/*`. Literal segments (`mine`, `invitations`, `memberships`) are
registered before `/api/initiatives/:id`, and `/photos/order` before
`/photos/:photoId`.

`POST` and `PATCH` accept only `name`, `address`, `cep`, `city`, `state`,
`email_owner`, `acting_area`, `impact`, `type`, `main_ods`, `lat`, `lon`,
`description` (up to 2000 characters), `whatsapp`, `instagram` and `website`;
other body fields such as `owner`, `points`, or the suspension fields are
ignored. `PATCH` changes exactly the fields present in the body, zero values
included, and returns the full initiative. Responses expose `description`,
`logo_url`, `whatsapp`, `instagram` and `website`; `GET /:id` also returns
`viewer_role` (`owner`, `member` or `null`) and `members_count`.

The contact fields are optional; an empty string or `null` clears them, and
they are stored normalized:

- `whatsapp`: digits only, with `55` prepended to a 10 or 11 digit number
  (DDD + number); `(11) 98765-4321` becomes `5511987654321`. Anything that does
  not end up as 12 or 13 digits starting with `55` answers
  `400 {"code":"invalid_whatsapp"}`.
- `instagram`: the handle without `@`, taken from `handle`, `@handle`,
  `instagram.com/handle` or `https://www.instagram.com/handle/?...`; it must
  match `^[A-Za-z0-9._]{1,30}$`, else `invalid_instagram`.
- `website`: `https://` is prepended when there is no scheme; it must be an
  absolute `http`/`https` URL without spaces or user info, whose host contains a
  dot, up to 255 characters, else `invalid_website`.

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
- Photos follow their initiative: `GET /:id/photos` answers
  `initiative_not_found` for a suspended initiative unless the requester owns
  it, and the public image answers `404 {"code":"photo_not_found"}` while the
  initiative is suspended.

Members join by invitation: the owner invites a registered user by e-mail and
the invitee accepts or declines; a member can leave at any time. Deleting an
initiative also deletes its members, photos and logo.

Photos are listed as `{"id", "url", "caption", "position"}`, ordered by
`position`; position `0` is the cover. Only the owner adds, captions, reorders
and deletes them. An initiative holds up to 12 photos; another upload answers
`409 {"code":"photo_limit_reached"}`. Captions are trimmed, up to 200
characters, and an empty caption becomes `null`. `PATCH` takes
`{"caption": string|null}`; `PUT /photos/order` takes `{"photo_ids": [...]}`,
which must list every current photo exactly once (else `invalid_request`), and
returns the reordered list. Deleting a photo moves the following ones back one
position. Images follow the logo rules below.

The older aliases (`/createOne`, `/getAllOnes`, `/getOneById/:id`,
`/updateOne/:id`, `/deleteOne/:id`) share these handlers and rules.

## Image storage

Logos and photos live in Google Cloud Storage, in the bucket named by
`STORAGE_BUCKET`, as `initiatives/<id>/logo-<hash>.<ext>` and
`initiatives/<id>/photos/<photoId>-<hash>.<ext>`. Uploads are PNG, JPEG or WebP
images up to 2 MiB (`image_too_large`, `invalid_image`). The public image
routes send an `ETag` and are cached for good when the URL's `v` matches the
current hash. On Cloud Run the client uses the service account through
Application Default Credentials; locally it talks to the emulator set by
`STORAGE_EMULATOR_HOST`. Without `STORAGE_BUCKET` the service still starts and
the image routes answer `503 {"code":"storage_unavailable"}`.

## Configuration convention

Use `DB_SSLMODE` in all new configuration. The application temporarily accepts
the older `DBSSLMode` spelling to avoid breaking existing developer machines.
