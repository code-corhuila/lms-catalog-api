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
│   └── usecase/                 → CreateBook (HU-04), UpdateBook (HU-09), LoanBookCopy/ReturnBookCopy (HU-06/HU-07)
├── config/                      → environment variable loading
├── adapter/
│   ├── in/httpapi/               → chi router, middleware, handlers, response envelope
│   └── out/
│       └── persistence/           → BookRepository and IdempotencyStore, both against PostgreSQL
└── infrastructure/logger/        → structured (zap) logger — not a port implementation
```

## Authentication

Two algorithms, each with its own key (`rules/2-anexos/C-api-hexagonal.md`, numeral 5.3.7):
**RS256**, verified with `lms-access-api`'s public key (`JWT_PUBLIC_KEY`), for a real
Administrator session; **HS256**, verified with a separate `INTERNAL_JWT_SECRET`, for the tokens
`lms-circulation-api` mints to call this service's `/books/{id}/loan-copy` and `/return-copy` —
this service only validates, it never mints one itself. Never the same key for both — see
`internal/adapter/in/httpapi/middleware/auth.go`'s doc comment for why.

## Known gaps against `rules/2-anexos/C-api-hexagonal.md`

- **Idempotent creation is durable now, not provisional.** `POST /books` honors
  `Idempotency-Key` against `catalog.idempotency_key` (`lms-catalog-db`).
- **`GET /books` now exists** (`SearchBooks` use case, same offset-pagination envelope as
  `lms-membership-api`'s `GET /students` — `meta` echoing `page`/`limit`/`totalPages`). Was a
  declared gap; closed after local end-to-end testing showed `lms-catalog-portal`'s
  `BooksListPage` had nothing to call.
- **`PATCH /books/{id}` now exists** (`UpdateBook` use case, HU-09 — title, author, category,
  year; ISBN and copy counts are not editable). Was missing: `lms-catalog-portal`'s edit row
  called it and got 404, shown as "Something went wrong".

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
