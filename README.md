# lms-catalog-api

> Catalog bounded context: search, records, copies

Part of the **LMS Library** distributed system — team `lms-library`, Grupo 2.
Governance and documentation live in [`library-docs`](https://github.com/code-corhuila/library-docs).

## Migration scope

**Comes from** `lms-library` → `catalog-service/`: `cmd/`, `internal/{domain,application,infrastructure}`,
`go.mod`, `go.sum`, `Dockerfile`, `Makefile`.

**Without `migrations/`** — the schema moves to `lms-catalog-db`.

Keep the hexagonal layout already in place: domain with its ports and tests, use cases in
`application`, adapters in `infrastructure`.

The full map lives in `library-docs`.

## Structure

Follows `rules/2-anexos/C-api-hexagonal.md` (the course's own repository norm) — see
`ADR-010-liquibase-for-database-migrations.md` for the related `-db` decision.

```
cmd/api/                       → entry point (main.go), the composition root
internal/
├── domain/catalog/              → Book aggregate — no port, no framework import
├── application/
│   ├── port/
│   │   ├── in/                  → inbound ports the HTTP adapter depends on (book_usecases.go)
│   │   └── out/                 → outbound ports the use cases depend on (ports.go)
│   └── usecase/                 → CreateBook (HU-04), LoanBookCopy/ReturnBookCopy (HU-06/HU-07)
├── config/                      → environment variable loading
├── adapter/
│   ├── in/httpapi/               → chi router, middleware, handlers, response envelope
│   └── out/
│       ├── persistence/           → BookRepository against PostgreSQL
│       └── idempotency/             → provisional in-memory IdempotencyStore (see below)
└── infrastructure/logger/        → structured (zap) logger — not a port implementation
```

## Known gaps against `rules/2-anexos/C-api-hexagonal.md`

- **Idempotent creation is provisional.** `POST /books` honors `Idempotency-Key`, but
  `internal/adapter/out/idempotency` is an in-memory map — it does not survive a restart and
  does not coordinate across more than one running instance. The durable version needs
  `lms-catalog-db`'s own `idempotency_key` table, which doesn't exist yet
  (`ADR-010-liquibase-for-database-migrations.md`).
- **Auth stays HS256/shared-secret, not RS256/public-key.** Switching needs a coordinated change
  with `lms-access-api` (the token issuer) — tracked separately.
- **No `GET /books` collection endpoint yet** — already a declared gap in
  `library-docs/07-api/guidelines.md`, so the pagination fix this norm otherwise requires (`meta`
  echoing `page`/`limit`/`totalPages`) has nothing to apply to in this service today.

---

## Branching

Three permanent branches. **None of them accepts a direct commit** — you enter through a child
branch and leave through a Pull Request.

```
develop  <--PR--  feat/... fix/... chore/...
qa       <--PR--  qa/...
main     <--PR--  release/...  hotfix/...
```

Promotion happens **by re-application** (`git cherry-pick -x`), never by merging one permanent
branch into another: `merge develop -> qa` and `merge qa -> main` do not exist in this model.

`main` requires **1 approval from `ariel5253`**. On `develop` and `qa` the team sets its own review
rule.

Full policy: `00-governance/branching-policy.md` in `library-docs`.
