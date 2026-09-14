---
layout: default
title: "Limits"
---

# Limits

pdfcpu uses limits to bound individual input-driven allocations and network requests. They are not a total memory,
CPU-time or disk-space budget for a command. A document can contain many objects, streams and images within their
individual limits.

The settings below follow their order in the default `config.yml`. Defaults refer to v0.16.0.
Use [`pdfcpu config inspect`](/config/config_inspect) to see effective configured values and
[`pdfcpu config validate`](/config/config_validate) after editing the file. Inspection does not include per-operation
command-line overrides such as `--offline`.

## Settings in config.yml order

This excerpt contains the network policy and limit settings in their original order.<br>
it is not a complete configuration.

```yaml
offline: false
timeout: 5
timeoutCRL: 10
timeoutOCSP: 10
allowedRevocationHosts: []
preferredCertRevocationChecker: crl
formFieldListMaxColWidth: 0
maxInputBytes: 0
maxObjectBytes: 64 MB
maxStreamBytes: 512 MB
maxDecodeBytes: 512 MB
maxImagePixels: 100 MP
maxImageBytes: 512 MB
```

Byte limits accept an integer byte count or an integer followed by a unit, such as `64 MB`. `KB`/`KiB`, `MB`/`MiB`
and `GB`/`GiB` use powers of 1024; `64 MB` means 67,108,864 bytes. Pixel limits accept an integer count or `MP`, where
`1 MP` means 1,000,000 pixels. Use a space before the unit. Unit names are case-insensitive; fractional quantities
such as `1.5 MB` are not accepted.

Except for the two documented zero values below, YAML limits and timeouts must be positive. Omitted resource-limit
keys retain their built-in defaults; zero does not generally disable a limit.

## Offline network policy

**`offline` — default: `false`.**

Set `offline: true` to disable outbound network activity. Remote images are skipped during JSON creation and form
filling, link validation makes no HTTP requests, and certificate validation performs no live CRL or OCSP requests.
The checks run before network access, including DNS resolution. The CLI `--offline` flag overrides this setting for
that operation.

## HTTP timeout

**`timeout` — default: `5` seconds.**

Bounds HTTP requests using the general HTTP timeout, including remote image retrieval and link validation.
Set a positive integer number of seconds. This is not a deadline for the complete PDF command.

## CRL timeout

**`timeoutCRL` — default: `10` seconds.**

Bounds HTTP requests for certificate revocation lists. Set a positive integer number of seconds.
Multiple requests can occur during certificate validation; this is not a total validation-time budget.
Offline mode prevents live requests regardless of the timeout.

## OCSP timeout

**`timeoutOCSP` — default: `10` seconds.**

Bounds HTTP requests for Online Certificate Status Protocol responses. Set a positive integer number of seconds.
As with the CRL timeout, it applies to requests rather than the entire validation operation, and does not enable
network access in offline mode.

## Allowed revocation hosts

**`allowedRevocationHosts` — default: `[]`.**

Lists private hosts explicitly trusted for CRL and OCSP requests, for example `[ocsp.example.corp, crl.example.corp]`.
This is a revocation-network policy setting, not a numerical limit or a general HTTP allowlist. An empty list does
not mean all public revocation requests are disabled; use `offline` to disable network access.

## Preferred revocation checker

**`preferredCertRevocationChecker` — default: `crl`.**

Selects `crl` or `ocsp` as the preferred certificate revocation-checking mechanism. The CRL and OCSP timeouts remain
separate. This preference is included here to preserve the configuration-file sequence; it is not a resource limit.

## Form-list column width

**`formFieldListMaxColWidth` — default: `0` (no configured width limit).**

Limits the displayed `AltName`, `Default` and `Value` columns in `pdfcpu form list` when greater than zero.
Use a nonnegative integer. This changes listing presentation, not the stored field contents or the amount of PDF
input that may be processed.

## PDF input limit

**`maxInputBytes` — default: `0` (unlimited).**

Limits the byte length of each PDF input. Set a positive byte count or a quantity such as `512 MB` to enable it.
Negative values are invalid. Text inspection shows `unlimited` for zero; JSON inspection reports numeric `0`.

Oversized seekable PDFs are rejected before parsing. For stdin, pdfcpu writes at most the configured limit to its
input spool, then probes one additional byte to detect overflow. An input exactly at the limit is accepted after EOF
is confirmed. On overflow, the temporary input is removed and an existing output destination is preserved.
Cancellation during input copying remains cooperative, including while awaiting EOF.

The limit applies separately to every PDF read, including each merge source and generated PDFs read again during
multi-fill merging or output validation. It does not add together merge sources or bound generated output size,
auxiliary JSON/CSV/image/font inputs, total temporary storage or process memory. Allow separate storage for output staging.

Go callers set `Configuration.Limits.MaxInputBytes` and can recognize overflow with
`errors.Is(err, model.ErrInputSizeLimit)`. Zero also means unlimited in the Go API.

## Object buffer limit

**`maxObjectBytes` — default: `64 MB` (67,108,864 bytes).**

Limits the reader's buffer for one indirect object, including its object header and end marker, or the stream
dictionary and opening stream delimiter. It covers ordinary indirect objects and cross-reference stream dictionaries.
Existing schema-1 files that omit this setting retain the default.

It does not limit stream payloads, decoded objects inside object streams, total PDF size or process memory.
Stream payloads use the separate encoded and decoded stream limits below. Reconstruction retains an independent
64 MiB line-scanner cap.

Go callers set `Configuration.Limits.MaxObjectBytes`; zero selects the default. In YAML, use a positive value.

## Encoded stream limit

**`maxStreamBytes` — default: `512 MB` (536,870,912 bytes).**

Limits encoded stream bytes read from a PDF. This is the stored stream payload before filter decoding, not the
complete PDF or the indirect-object dictionary surrounding the stream.

A compressed stream can fit this limit but exceed `maxDecodeBytes` when expanded. Configure both limits for the
inputs you intend to accept. Go callers set `Configuration.Limits.MaxStreamBytes`.

## Decoded stream limit

**`maxDecodeBytes` — default: `512 MB` (536,870,912 bytes).**

Limits decoded stream bytes produced by filters on the bounded decoding paths. This controls expansion of
compressed stream data independently of its encoded size. It is not an aggregate allowance shared by all streams.

Go callers set `Configuration.Limits.MaxDecodeBytes`. Image dimension and buffer checks also apply when image
processing requires them; passing this limit does not imply an image will fit those limits.

## Image pixel limit

**`maxImagePixels` — default: `100 MP` (100,000,000 pixels).**

Limits the product of image width and height for decoded/rendered image dimensions. It is a pixel count, not a
maximum width or height independently. For example, an image measuring 10,000 by 10,000 pixels reaches the default
pixel limit exactly, but must still satisfy the image-byte and other applicable limits.

Go callers set `Configuration.Limits.MaxImagePixels`.

## Image buffer limit

**`maxImageBytes` — default: `512 MB` (536,870,912 bytes).**

Limits decoded/rendered image buffer sizes. A small compressed image file can require a large buffer after decoding.
The checked size depends on the image representation; rendering checks include a four-bytes-per-pixel buffer.
This is separate from `maxImagePixels` and does not bound the total memory used by every image in a document.

Go callers set `Configuration.Limits.MaxImageBytes`.

## Additional limits shown by inspection

`config inspect` also reports the following structural limits. They are not keys in the v0.16.0 `config.yml` schema;
do not add their inspection names to the YAML file. Go callers configure the corresponding `Configuration.Limits`
fields. They are listed here in inspection order, after the YAML limits above.

| Inspection field | Go field | Default | Scope |
|:-----------------|:---------|:--------|:------|
| `max object count` | `MaxObjectCount` | 10,000,000 | Cross-reference stream `/Size` expansion and object-number ranges. This is not a total count of every object processed by a command. |
| `max object stream count` | `MaxObjectStreamCount` | 1,000,000 | The `/N` count of objects in one object stream. |
| `max object stream first` | `MaxObjectStreamFirst` | 16 MiB (16,777,216 bytes) | The `/First` offset defining the prolog before object data in an object stream. |
| `max xref entries` | `MaxXRefEntries` | 10,000,000 | Cross-reference stream `/Index` expansion, including the default range when `/Index` is absent. |
| `max recursion depth` | `MaxRecursionDepth` | 100 | Nesting on guarded parser and object-graph traversal paths. This is not a bound on total traversal work or all command execution. |

For Go configuration, start with `model.NewDefaultConfiguration()` and change the fields you need. Do not treat a
zero-filled `ResourceLimits` struct as a request for unlimited operation: zero-value behavior is field-specific.
