# AGENTS.md — fake-sefaz

Rules for AI agents in this repository. Raise any conflict between a request and
a rule before writing code. This file holds rules only, never feature
descriptions.

## Product

- Stand-in for the SEFAZ web services in development and tests of Vender+. Never
  deployed where a real fiscal document could reach it; never claims legal
  validity.
- Fidelity is the point: elements, their order, `cStat` and `xMotivo` follow the
  Manual de Orientação do Contribuinte. A rule that cannot be reproduced
  faithfully is left out and written in the README, never approximated.

## Stack

- Go latest stable, standard library only; `go.mod` has no `require` block. A
  dependency needs a capability the standard library lacks entirely, named in
  the PR.
- XSD validation is implemented in `internal/xsd` (Go bindings are cgo over
  libxml2 and would end the static build). Extend its subset only when a
  published schema needs it, naming the schema in the PR.

## Rules

1. English for code, identifiers, file names, commits and docs.
2. No comments in code. No duplicated code: extract on the second occurrence.
3. Small functions, one responsibility, early returns, no deep nesting, no
   abbreviations.
4. A `cStat` is never invented: only codes of the official catalogue, from
   `internal/status`.
5. State is in memory and every access goes through `internal/store`, which owns
   the mutex.

## Structure

```
cmd/fakesefaz/main.go   wiring and startup only
internal/
  config/      environment        dfe/        access key, models, parser, events
  uf/          states, cUF        status/     cStat catalogue and wording
  soap/        envelopes, gzip    xsd/        schemas and validation
  authorizer/  every rule         nfe/        typed docs, models 55 and 65
  dfews/       table driven 57, 67, 58, 63, 66
  sat/         CF-e SAT, model 59 store/      documents, batches, protocols
  scenario/    forced outcomes    server/     routing, SOAP handler, admin API
```

- Direction: `server` -> model package -> `authorizer` -> `store`. A model package
  never imports `net/http`; `authorizer` never imports a model package; `store`
  imports neither.
- Every rule that decides a `cStat` lives in `authorizer` only. A model package
  maps XML to `authorizer.Submission` and the result back.
- NF-e keeps typed structs; other SOAP models share the table driven
  implementation through `dfe.ModelSpec`.
- New SOAP model: an entry in `internal/dfe.modelSpecs` (namespace, version,
  tags) with its events in `internal/dfe.eventCatalogue`, and one in
  `internal/dfews.operations`. Nothing else, unless its access key breaks the
  NF-e layout (like CF-e SAT), which needs a branch in `ParseAccessKey`.
- Packages: short, lowercase, one word, no plurals. Files `snake_case.go`, one
  operation per file in `internal/nfe`. Acronyms keep their case: `ufCode`,
  `HTTPStatus`, `NSU`.

## Schemas

- XSD packages are downloaded (`make schemas` into the git-ignored `schemas/`),
  never committed. `FAKE_SEFAZ_SCHEMA_DIR` turns validation on; with it unset the
  service still works, and an unknown root is served, not refused.
- `internal/xsd/testdata` holds small schemas written for the tests, not copies
  of official ones.

## Tests

- Every operation is covered end to end through `httptest` in `internal/server`
  with real SOAP envelopes, asserting the `cStat`. A rejection rule is not done
  until a test triggers it. The clock is injected; no test sleeps.
- Before touching the validator, also run it against the published packages:
  `FAKE_SEFAZ_SCHEMA_DIR=$PWD/schemas go test ./...`

## Commits

- Always through the `commit` skill, never a hand written `git commit`. End every
  prompt that changed files with it, before answering.
- The checkout is shared by several sessions: commit only what you wrote, stage by
  path or hunk, and say what was left behind.
- One commit per purpose, Conventional Commits prefix, never `git add -A`, never
  push unless asked.
