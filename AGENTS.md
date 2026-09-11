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

XSD validation is the one capability the standard library really lacks, and it
still did not earn a dependency: every Go binding is cgo over libxml2, which
would end the static build. `internal/xsd` implements the subset these schemas
use instead. Extend that subset only when a published schema starts using
something it does not cover, and say which schema in the pull request.

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
  dfe/                     Access key, model vocabulary, generic parser, events
  uf/                      The 27 states, cUF codes and authorizers
  status/                  cStat catalogue and its per model wording
  soap/                    Envelope decoding and encoding, gzip payloads, faults
  xsd/                     Schema loading and instance validation
  authorizer/              Every rule: authorization, events, voiding
  nfe/                     Typed documents for models 55 and 65
  dfews/                   Table driven documents for 57, 67, 58, 63 and 66
  sat/                     CF-e SAT equipment commands for model 59
  store/                   Documents, batches, voidings, protocol sequence
  scenario/                Forced outcomes and service state
  server/                  Routing, SOAP handler, admin API
```

Dependency direction: `server` -> a model package -> `authorizer` -> `store`.
A model package never imports `net/http`, `authorizer` never imports a model
package, and `store` never imports either.

Every rule that decides a `cStat` lives in `authorizer` and nowhere else. A
model package only maps XML to `authorizer.Submission` and the result back to
its own response document. When a rule has to change, it changes once.

NF-e keeps typed structs because its contract is the richest one: a batch, a
receipt, a registration query and the distribution service. The other SOAP
models share one table driven implementation, because their documents are the
same skeleton under different tag names, which `dfe.ModelSpec` carries.

## 5. File and package naming

- Package names: short, lowercase, single word, no underscores, no plurals.
- File names: `snake_case.go`, one operation per file inside `internal/nfe`.
- Exported identifiers `PascalCase`, unexported `camelCase`, acronyms keep
  their case: `ufCode`, `HTTPStatus`, `NSU`.

## 6. Adding a document model

The access key, the store, the protocol sequence and every rule are already
model agnostic. A new SOAP model needs two things and nothing else:

1. An entry in `internal/dfe.modelSpecs` with its namespace, version and tag
   names, and its events in `internal/dfe.eventCatalogue`.
2. An entry in `internal/dfews.operations` naming its request roots, its
   response documents and its web service names.

Only a model whose access key does not follow the NF-e field layout, the way
CF-e SAT does not, needs a branch in `ParseAccessKey`.

## 7. Schemas

The XSD packages are a download, never a commit. `make schemas` fetches them
into `schemas/`, which is ignored by git, and `FAKE_SEFAZ_SCHEMA_DIR` turns
validation on. The service has to keep working with the variable unset, so a
schema is never a precondition for an operation: an unknown root is served, not
refused.

`internal/xsd/testdata` holds small schemas written for the tests. They exist so
the suite runs with no download, and they are not copies of anything official.

## 8. Tests

Every operation is covered end to end through `httptest` in
`internal/server`, driving real SOAP envelopes and asserting on the returned
`cStat`. A new rejection rule is not done until a test drives the request that
triggers it. The clock is injected; no test sleeps.

`internal/xsd` is tested twice: against `testdata`, always, and against the
published packages when `FAKE_SEFAZ_SCHEMA_DIR` points at them, skipped
otherwise. Run the second one before touching the validator:

```
FAKE_SEFAZ_SCHEMA_DIR=$PWD/schemas go test ./...
```

## 9. Commits

Commits are made through the `commit` skill, never with a hand written
`git commit`. Every prompt ends with it: once the work the prompt asked for is
finished, and before answering, invoke the skill whenever the working tree has
changes. A prompt that changed no file has nothing to commit. The same holds
when the request is only "commit".

Several sessions share this checkout, so a session commits only what it wrote.
Changes another session left in the tree stay out of the commits, staged around
by path or by hunk, and the answer says what was left behind.

The skill owns the details, and it overrides habit where the two disagree: one
commit per purpose, staged path by path rather than with `git add -A`, and a
Conventional Commits prefix on every subject. Never push unless asked.
