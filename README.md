# fake-sefaz

A local stand-in for the SEFAZ web services, written in Go with the standard
library only. It answers the NF-e 4.00 SOAP contract so a fiscal document can be
authorized, queried, cancelled and voided without an ICP-Brasil certificate, a
CNPJ or an internet connection.

It is a development and test dependency. It authorizes nothing that has legal
value and it does not validate signatures cryptographically.

## Running

```
go run ./cmd/fakesefaz
```

It listens on `:8080` by default. Configuration comes from the environment:

| Variable | Default | Meaning |
| --- | --- | --- |
| `FAKE_SEFAZ_ADDRESS` | `:8080` | Listen address |
| `FAKE_SEFAZ_CANCELLATION_WINDOW` | `24h` | Cancellation deadline for model 55 |
| `FAKE_SEFAZ_CANCELLATION_WINDOW_NFCE` | `30m` | Cancellation deadline for model 65 |
| `FAKE_SEFAZ_MAX_DISTRIBUTION_DOCUMENTS` | `50` | Documents per `distDFeInt` answer |

## Pointing a client at it

The operation is resolved from the root element of the SOAP body, never from
the path or from `SOAPAction`. Any URL works, so a client can keep the shape of
the real address it already has configured:

```
http://localhost:8080/homologacao/SP/NFeAutorizacao4
http://localhost:8080/producao/RS/NFeStatusServico4
http://localhost:8080/anything/at/all
```

When the path carries a state acronym or an environment word (`homologacao`,
`producao`), they are used to fill `cUF` and to reject a document whose `tpAmb`
disagrees with the environment being addressed. `GET /admin/endpoints` prints
the full URL catalogue for the 27 states plus the national environment.

A request may also be posted bare, without the SOAP envelope, in which case the
answer comes back bare as well.

## Web services

| Operation | Web service | Answer |
| --- | --- | --- |
| `consStatServ` | NFeStatusServico4 | `retConsStatServ` |
| `enviNFe` | NFeAutorizacao4 | `retEnviNFe`, synchronous or with a receipt |
| `consReciNFe` | NFeRetAutorizacao4 | `retConsReciNFe` |
| `consSitNFe` | NFeConsultaProtocolo4 | `retConsSitNFe` with the events attached |
| `inutNFe` | NFeInutilizacao4 | `retInutNFe` |
| `envEvento` | NFeRecepcaoEvento4 | `retEnvEvento` |
| `ConsCad` | CadConsultaCadastro4 | `retConsCad` |
| `distDFeInt` | NFeDistribuicaoDFe | `retDistDFeInt` with gzipped `docZip` |

Events: cancellation, cancellation by substitution, correction letter and the
four recipient acknowledgements.

## Rules that produce a rejection

The service keeps state, so the rejections that matter for a real emission
client happen on their own:

- `236` access key check digit does not match
- `226` `cUF` unknown or different from the one inside the access key
- `252` `tpAmb` different from the environment addressed by the URL
- `297` document without a signature
- `693` homologation document whose recipient name is not the required text
- `204` access key already authorized, or number already voided
- `539` same issuer, model, series and number under a different access key
- `217` query or event for a document that was never authorized
- `573` event repeated for the same key, type and sequence
- `501` cancellation past the deadline for the model
- `106` receipt not found, `105` batch still processing

## Forcing an outcome

Two ways, both meant for tests.

A header on a single request:

```
X-Fake-Sefaz-Status: 539
```

Or a rule through the admin API, which survives until it is consumed or removed:

```
curl -X POST localhost:8080/admin/scenarios \
  -d '{"operation":"enviNFe","issuerTaxId":"99999999000191","status":301,"remaining":1}'
```

A rule matches on any combination of `operation`, `issuerTaxId`, `model` and
`keySuffix`. `remaining` makes it expire after that many hits; leave it out for
a rule that never expires.

## Admin API

| Method and path | Purpose |
| --- | --- |
| `GET /admin/health` | Liveness and document count |
| `GET /admin/endpoints` | URL catalogue per state and environment |
| `GET /admin/units` | The 27 states, their `cUF` and their authorizers |
| `GET /admin/models` | Document models and which ones are implemented |
| `GET /admin/status-codes` | The `cStat` catalogue this service can answer |
| `GET /admin/documents` | Authorized documents, filtered by `issuer`, `environment`, `model`, `afterNsu` |
| `GET /admin/documents/{key}` | One document with its XML and its events |
| `GET /admin/voidings` | Voided number ranges |
| `GET /admin/service`, `PUT /admin/service` | Read or set `status`, `asynchronous`, `averageTime` |
| `GET`, `POST`, `DELETE /admin/scenarios` | Rule management |
| `DELETE /admin/state` | Wipe documents, batches, voidings and rules |

Pausing the service is how a contingency path gets exercised:

```
curl -X PUT localhost:8080/admin/service -d '{"status":108}'
```

Making the authorizer answer asynchronously, so the client has to poll the
receipt, is the same call with `{"asynchronous":true,"averageTime":3}`.

## What is not here

- Models other than 55 and 65. `dfe.Models` already carries CT-e, CT-e OS,
  MDF-e, CF-e, BP-e and NF3e, and the store and the access key are model
  agnostic, but their web services and response documents are not written.
- NFS-e, which is municipal and follows a different standard altogether.
- XSD validation. A malformed document is answered with `215` or `225` only
  when it fails to unmarshal, not because a schema was checked.
- Signature verification. `297` is answered for a missing signature, never for
  a wrong one.
- Persistence. State lives in memory and dies with the process.
