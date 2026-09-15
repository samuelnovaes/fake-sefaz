# fake-sefaz

A local stand-in for the SEFAZ web services, written in Go with the standard
library only. It answers the SOAP contracts of every electronic fiscal document
the SEFAZ authorizes, plus the CF-e SAT equipment commands, so a document can be
authorized, queried, cancelled and voided without an ICP-Brasil certificate, a
CNPJ or an internet connection.

| Model | Document | Transport |
| --- | --- | --- |
| 55 | NF-e | SOAP |
| 65 | NFC-e | SOAP |
| 57 | CT-e | SOAP |
| 67 | CT-e OS | SOAP |
| 58 | MDF-e | SOAP |
| 63 | BP-e | SOAP |
| 66 | NF3e | SOAP |
| 59 | CF-e SAT | local equipment commands over JSON |

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
| `FAKE_SEFAZ_SUBSTITUTION_WINDOW` | `168h` | Deadline of the NFC-e cancellation by substitution, counted from the authorization |
| `FAKE_SEFAZ_OFFLINE_DEADLINE` | `24h` | Delay after `dhEmi` past which an NFC-e issued in contingency is answered `150` |
| `FAKE_SEFAZ_MAX_DISTRIBUTION_DOCUMENTS` | `50` | Documents per `distDFeInt` answer |
| `FAKE_SEFAZ_SCHEMA_DIR` | empty | Directory with the XSD packages; empty turns schema validation off |

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

NF-e and NFC-e:

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

CT-e and CT-e OS:

| Operation | Web service | Answer |
| --- | --- | --- |
| `CTe` | CTeRecepcaoSincV4 | `retCTe` with `protCTe` |
| `CTeOS` | CTeRecepcaoOSV4 | `retCTe` with `protCTe` |
| `consSitCTe` | CTeConsultaV4 | `retConsSitCTe` with `procEventoCTe` |
| `consStatServCte` | CTeStatusServicoV4 | `retConsStatServCte` |
| `eventoCTe` | CTeRecepcaoEventoV4 | `retEventoCTe` |

MDF-e:

| Operation | Web service | Answer |
| --- | --- | --- |
| `MDFe` | MDFeRecepcaoSinc | `retMDFe` with `protMDFe` |
| `consSitMDFe` | MDFeConsulta | `retConsSitMDFe` |
| `consStatServMDFe` | MDFeStatusServico | `retConsStatServMDFe` |
| `eventoMDFe` | MDFeRecepcaoEvento | `retEventoMDFe` |
| `consMDFeNaoEnc` | MDFeConsNaoEnc | `retConsMDFeNaoEnc` |

BP-e and NF3e follow the same five-operation shape: `BPe`/`NF3e` for the
synchronous authorization, `consSitBPe`/`consSitNF3e`, `consStatServBPe`/
`consStatServNF3e` and `eventoBPe`/`eventoNF3e`.

`distDFeInt` is served once and answers for every model, since it filters by
CNPJ and environment and the store holds all of them.

Business rules run after the schema, not instead of it. A document can be
schema perfect and still be refused: a real NFC-e whose access key check digit
does not match its own `cDV` passes every XSD and is answered `236`.

Events per model:

| Model | Events |
| --- | --- |
| NF-e | correction letter, cancellation and the four recipient acknowledgements |
| NFC-e | the NF-e events plus cancellation by substitution |
| CT-e, CT-e OS | correction letter, cancellation, EPEC, delivery receipt, service disagreement |
| MDF-e | cancellation, closing, driver added, document added |
| BP-e | cancellation, boarding missed, seat changed |
| NF3e | cancellation |
| CF-e SAT | cancellation, through `CancelarUltimaVenda` |

A payload that arrives gzipped and base64 encoded inside a `*DadosMsg` element,
the way MDF-e and NF3e clients send it, is inflated before the operation is
resolved.

## CF-e SAT

Model 59 is not a web service: the SAT equipment issues the document locally and
the application talks to it through a library, not over SOAP. The equipment is
mocked as a JSON surface with the same commands and the same pipe delimited
return string:

```
curl -X POST localhost:8080/sat/EnviarDadosVenda \
  -d '{"numeroSessao":12345,"dadosVenda":"<CFe>...</CFe>"}'
```

The answer carries `retorno` verbatim, the same string already split into
`campos`, and, for a sale, the minted access key. When the CF-e arrives without
an `Id`, the equipment mints the 44 digit key the way a real SAT does, using the
CF-e layout, which differs from the NF-e one: nine digits of equipment serial
and six of document number. The document then lands in the same store as every
other model, so `/admin/documents` lists it and `CancelarUltimaVenda` cancels it.

`GET /admin/sat/commands` lists the fifteen commands and their success codes.

## Schema validation

Point the service at the published XSD packages and every request is validated
against them before any rule runs. The packages are a download, not part of this
repository:

```
make schemas
FAKE_SEFAZ_SCHEMA_DIR=$PWD/schemas go run ./cmd/fakesefaz
```

The script pulls the NF-e, CT-e and MDF-e packages into `schemas/`, one folder
each. Loading them logs how much was indexed:

```
msg="schemas loaded" documents=282 roots=147
```

A request whose root element has a schema is validated against the version the
`versao` attribute asks for. `enviNFe` answers `225`, every other message
answers `215`, and the offending path is logged:

```
schema="enviNFe/NFe/infNFe/ide/tpAmb: valor \"7\" fora da lista permitida"
```

A root with no schema is served as usual, so a partial download only narrows
what is checked. `GET /admin/schemas` says what is loaded.

The validator is written against the subset of XSD these schemas actually use,
which is a narrow one: `sequence`, `choice`, `element` including `ref`, `any`,
`attribute`, `anyAttribute`, `simpleContent` with `extension`, `include`,
`import`, and the facets `pattern`, `enumeration`, `length`, `minLength`,
`maxLength`, `minInclusive`, `maxInclusive` and `whiteSpace`. Several patterns
inside one `restriction` are alternatives, as the specification says, while
patterns along a derivation chain all have to pass. Patterns are compiled with
`regexp`, and all 126 in the NF-e 4.00 package compile.

What it does not do: `xs:group`, `xs:all`, substitution groups, complex type
derivation, `xs:list`, `xs:union`, `totalDigits`, `fractionDigits`, and the
identity constraints `unique`, `key` and `keyref`. None of them appear in the
fiscal schemas. A wildcard is always treated as `processContents="skip"`.

## Rules that produce a rejection

The service keeps state, so the rejections that matter for a real emission
client happen on their own:

- `236` access key check digit does not match
- `226` `cUF` unknown or different from the one inside the access key
- `252` `tpAmb` different from the environment addressed by the URL
- `297` document without a signature
- `693` homologation document whose recipient name is not the required text
- `629` NF-e or NFC-e with `finNFe` 1 where an item's `qCom` times `vUnCom`,
  rounded half up to cents, differs from its `vProd` by more than R$ 0,01
  (MOC rule I11-10)
- `537` NF-e or NFC-e whose `ICMSTot/vDesc` differs from the sum of the items'
  `vDesc` by more than R$ 0,01 (MOC rule W10-10)
- `204` access key already authorized, `218` when that document is cancelled
  and `205` when it is denied (MOC Anexo I rules 2B08-20, 2B08-30, 2B08-40)
- `206` number already voided (rule 3B08-100)
- `539` same issuer, model, series and number under a different access key
- `150` instead of `100` for an NFC-e with `tpEmis` 4 or 9 received more than
  `FAKE_SEFAZ_OFFLINE_DEADLINE` after its `dhEmi` (rule B09-40, exception 2)
- `217` query or event for a document that was never authorized
- `562`, `561` and `613` for a query whose key is unknown while a document with
  the same issuer, model, series and number exists: different `cNF` (the stored
  key is appended as `[chNFe:...]`), different month, any other difference
  (MOC Visao Geral rules J03 to J06)
- `563` voiding a range already voided, answered with the earlier `nProt`;
  `256` a range overlapping a voided one; `241` a range holding a used number
  (rules I07, I07a, I08)
- `573` event repeated for the same key, type and sequence
- `501` cancellation past the deadline for the model
- `106` receipt not found, `105` batch still processing

Cancellation by substitution, event `110112`, exists for model 65 only; model
55 answers `215` for it. It follows table 5-38 of the Visao Geral:

- `920` the cancelled NFC-e was not issued with `tpEmis` 1
- `910` `chNFeRef` invalid (check digit, UF, year, month, CNPJ/CPF, model,
  number) and `911` incorrect (same key, UF, CNPJ/CPF, year-month outside the
  cancelled key's month or the one before it, model), with the field in xMotivo
- `573` repeated event, `494` cancelled NFC-e unknown
- `501` more than `FAKE_SEFAZ_SUBSTITUTION_WINDOW` after the authorization
- `580` cancelled NFC-e already cancelled or denied, `222` `nProt` differs
- `912` substitute unknown, `913` substitute denied or cancelled
- `914` substitute `dhEmi` more than two hours after the cancelled `dhEmi`
- `915` `vNF`, `916` `vICMS`, `917` recipient (`CNPJ`/`CPF`/`idEstrangeiro`,
  `IE`), `918` item count, `919` an item's `cProd`, `cEAN`, `xProd`, `NCM`,
  `CFOP`, `uCom`, `qCom`, `vUnCom`, `vProd` or `indTot` differ
- `921` substitute issued with `tpEmis` 1

An accepted substitution answers `135`, marks the NFC-e cancelled, and its
query answers `101` with `protNFe` and the `procEventoNFe`.

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

A rule matches on any combination of `operation`, `issuerTaxId`, `model`,
`keySuffix` and `issuance`, the `tpEmis` digit of the access key. Refusing only
the off-line retransmission:

```
curl -X POST localhost:8080/admin/scenarios \
  -d '{"operation":"enviNFe","issuance":"9","status":778}'
```

`remaining` makes it expire after that many hits; leave it out for a rule that
never expires.

`loseAnswer` processes the request normally, or with the forced `status` when
one is given, stores the outcome, and then drops the connection without writing
any HTTP response, the way a timeout after the SEFAZ processed the request looks
to the client. A later query of the key answers `100` with its protocol:

```
curl -X POST localhost:8080/admin/scenarios \
  -d '{"operation":"enviNFe","loseAnswer":true,"remaining":1}'
```

It applies to authorization, voiding and events of the SOAP models; the CF-e
SAT commands ignore it.

`656`, consumo indevido, refuses the whole request when forced, on
`retEnviNFe` or `retEnvEvento`, because the MOC blocks the web service rather
than one document.

## Admin API

| Method and path | Purpose |
| --- | --- |
| `GET /admin/health` | Liveness and document count |
| `GET /admin/endpoints` | URL catalogue per state and environment |
| `GET /admin/units` | The 27 states, their `cUF` and their authorizers |
| `GET /admin/models` | Document models, their XML vocabulary and their transport |
| `GET /admin/sat/commands` | CF-e SAT commands and their success codes |
| `GET /admin/schemas` | Whether schemas are loaded and which roots they cover |
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

- NFS-e, which is municipal and follows a different standard altogether. It does
  not belong in a stand-in for the state authorizers.
- Identity constraints and the exotic corners of XSD listed above. Everything
  the fiscal schemas use is checked; nothing else is.
- Signature verification. `297` is answered for a missing signature, never for
  a wrong one.
- Persistence. State lives in memory and dies with the process.
- Asynchronous authorization for the models other than NF-e and NFC-e. CT-e,
  MDF-e, BP-e and NF3e are answered synchronously, which is how their current
  layouts work anyway.
- The state specific contingency deadlines. The MOC gives 24 hours in rule
  B09-40 and the end of the first business day after the issuance in Anexo IV,
  both at the state's discretion; a single `FAKE_SEFAZ_OFFLINE_DEADLINE`
  counted from `dhEmi` stands for them, and no calendar of business days is
  kept. Rules B09-10 (`703`) and B09-40 (`704`) on the delay of `dhEmi` are not
  applied.
- The event rules that need data the service does not keep or check: author
  and certificate (`574`, `489`, `490`, `408`, `455`, `466`), taxpayer
  registry (`203`, `240`), event date (`577`, `578`, `579`), sequence limit
  (`594`), and the deferred extension request (`811`).
- The MDF-e specific pair of codes for the non-closed query. `MDFeConsNaoEnc`
  answers `138` and `137`, the generic located and not located codes, because
  no `cStat` is invented here.
