# AGENTS.md — fake-sefaz

Rules every AI agent must follow in this repository. When a request conflicts
with a rule here, raise the conflict before writing code.

## 1. Product

fake-sefaz stands in for the SEFAZ web services during development and tests of
Vender+. It is never deployed anywhere a real fiscal document could reach it,
and it never claims legal validity.

Fidelity to the real contract is the whole point. A response element, its
order, its `cStat` and its `xMotivo` follow the Manual de Orientacao do
Contribuinte. When a rule cannot be reproduced faithfully, it is left out and
written down in the README instead of being approximated.

## 2. Stack

Go, latest stable version, standard library only. There is no `require` block
in `go.mod` and there will not be one: SOAP here is XML over HTTP, and
`encoding/xml` plus `net/http` cover it. A dependency enters only for a
capability the standard library does not have at all, and the pull request has
to say which one.

## 3. Non-negotiable rules

1. All code, identifiers, file names, commit messages and documentation are in
   English.
2. No comments in code. If a block needs an explanation, rename things or
   extract a function until it explains itself.
3. No duplicated code. Extract a helper on the second occurrence.
4. Small functions, one responsibility, early returns, no deep nesting, no
   abbreviations in names.
5. A `cStat` is never invented. It comes from `internal/status`, which carries
   only codes that exist in the official catalogue.
6. State is in memory and every access goes through `internal/store`, which
   owns the mutex. No package reaches around it.

## 4. Project structure

```
cmd/fakesefaz/main.go      Wiring and startup only
internal/
  config/                  Environment loading
  dfe/                     Access key, document models, event types
  uf/                      The 27 states, cUF codes and authorizers
  status/                  cStat catalogue
  soap/                    Envelope decoding and encoding, faults
  nfe/                     Request and response documents, one file per operation
  store/                   Documents, batches, voidings, protocol sequence
  scenario/                Forced outcomes and service state
  server/                  Routing, SOAP handler, admin API
```

Dependency direction: `server` -> `nfe` -> `store`. The `nfe` package never
imports `net/http`, and `store` never imports `nfe`.

## 5. File and package naming

- Package names: short, lowercase, single word, no underscores, no plurals.
- File names: `snake_case.go`, one operation per file inside `internal/nfe`.
- Exported identifiers `PascalCase`, unexported `camelCase`, acronyms keep
  their case: `ufCode`, `HTTPStatus`, `NSU`.

## 6. Adding a document model

The access key, the store and the protocol sequence are already model
agnostic. A new model needs three things and nothing else:

1. Mark it implemented in `internal/dfe.modelSpecs`.
2. Add its request and response documents under `internal/nfe`, or a sibling
   package when the schema shares nothing with NF-e.
3. Register its operations in `webServices` and `endpoints`.

## 7. Tests

Every operation is covered end to end through `httptest` in
`internal/server`, driving real SOAP envelopes and asserting on the returned
`cStat`. A new rejection rule is not done until a test drives the request that
triggers it. The clock is injected; no test sleeps.
