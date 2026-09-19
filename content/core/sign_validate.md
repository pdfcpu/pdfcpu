---
layout: default
title: "Validate Signatures"
---

# Validate Signatures

Signature validation is under active development.
Support for additional signature formats, validation rules, and trust evidence is being expanded.

Validate signature integrity, report available trust evidence and perform a best-effort local assessment.

This command checks whether signed byte ranges still match their signatures and reports available signer, certificate, timestamp, revocation, DSS, and PAdES evidence.

Certificate-path and revocation output is based on pdfcpu's configured local certificate store and available revocation information.
pdfcpu performs technical signature authentication and reports each independently established result.<br><br>
It does not make enterprise-policy, legal-validity, eIDAS, qualified-signature, or full long-term-validation decisions.



```
pdfcpu signatures validate inFile [flags]
```

## Flags

| name   | description       | default | required |
|:-------|:------------------|:--------|:---------|
| a(ll)  | validate all signatures | false | no |
| f(ull) | detailed output   | false   | no |

## Certified and Authoritative Signatures

A certified signature is a special type of signature that locks the document at a certain point, allowing only certain permitted changes afterward.
It indicates that the document was approved in its original form by the certifying party.

An authoritative signature is the first signature encountered in the document when no certified signature is present.
pdfcpu uses it as the primary signature to inspect when no certified signature is present.

Any number of approval signatures may be applied after a certified signature.

By default, validation focuses only on the certified signature, if available, or otherwise the authoritative signature.
If `-all` is set, all signatures in the PDF are validated.

## Arguments

| name   | description                            | required |
|:-------|:---------------------------------------|:---------|
| inFile | PDF input file, use `-` to read from stdin | yes |

## Signature Types

PDF supports several types of signatures, each with a distinct purpose:

### Form Signature

A digital signature associated with a form field within the document.
It is primarily intended to authenticate the person who filled out the form and confirms the integrity of the entered data.

### Page Signature

A digital signature applied directly onto a page, often as an annotation or widget.
Its purpose is to authenticate the visible content of the page, ensuring that it has not been altered.

### Document Timestamp Signature

A signature based on an [RFC 3161](https://datatracker.ietf.org/doc/html/rfc3161) `TimeStampToken`.
A DTS is evidence that a document existed at a specific point in time, without binding it to a particular signer.
Usually associated with PDFs prepared for long term validation.

### Usage Rights Signature

A special signature used to enable extended features, such as form filling, commenting, and saving, in PDF viewers like Adobe Reader.
It also detects unauthorized changes that would invalidate these usage rights.
Has to be the only signature in the document.

## Summary of Signature Intentions

| type | intention | visibility |
|:-----|:----------|:-----------|
| Form Signature | Authenticate form data and signer identity | visible or invisible |
| Page Signature | Authenticate page content and appearance | visible or invisible |
| Document Timestamp Signature | Prove document existence at a point in time | invisible |
| Usage Rights Signature | Define locked features, detect tampering | invisible |

This is not intended as an in-depth introduction to PDF digital signatures.
For complete details, please refer to the [PDF 2.0 specification (ISO 32000-2:2020)](https://www.pdfa-inc.org/product/iso-32000-2-pdf-2-0-bundle-sponsored-access/).

It may not be immediately obvious whether a PDF contains signatures.
You can check for existing signatures using `pdfcpu info` on the command line.

## Validation Steps

### 1. Check Hash of Signed Bytes

Compare the hash from the signature with a computed hash to detect any document modifications.

### 2. Verify Crypto Signature

Check that the signature was created using the correct private key and matches the data.

### 3. Check Certificate Evidence

Check the signer certificate and report whether it chains up to a certificate in pdfcpu's configured local certificate store.
pdfcpu also reports certificate validity dates and performs best-effort revocation checks when suitable CRL or OCSP information is available.

`QC Policy` indicates whether the certificate contains a qualified-certificate policy identifier recognized by pdfcpu.
This is certificate inspection, not a legal, eIDAS, trusted-list or qualification conclusion.

These checks are useful for inspection and automation, but they are not a substitute for a dedicated trust policy, compliance profile, or legal-validity assessment.

### 4. Check Timestamp Evidence

Both document timestamps and embedded signature timestamps use RFC 3161 timestamp tokens, but they bind different data.
A document timestamp binds a PDF revision's signed byte ranges; an embedded signature timestamp binds the signer's
cryptographic signature value.

pdfcpu currently validates document timestamps within its supported local technical scope. This includes the CMS
signature, PDF message imprint, supported timestamp profile, TSA certificate, and the configured-local certificate path
and available revocation evidence.

Embedded signature timestamp tokens are currently located and parsed, including their generation time, but are not yet
authenticated. Their signature, message imprint, profile, and TSA certificate path therefore remain `unknown` in the
evidence report. This difference is a current implementation gap, not a trust-policy or product boundary.

## Checking Revocation

Certificates may be revoked for various reasons.
Checking the revocation status may require online access and depends on the certificate, the responder, and any embedded evidence in the PDF.
In detailed output, `Revocation: Local:  ...` identifies pdfcpu's best-effort local assessment; accompanying CRL and OCSP details report the available evidence.

You can configure timeout values for CRL and OCSP responders with:

* `timeoutCRL`
* `timeoutOCSP`

Revocation requests reject loopback, private, link-local, multicast, and unspecified addresses by default.
Use `allowedRevocationHosts` to explicitly allow private CRL or OCSP hosts that you trust, for example:

```yaml
allowedRevocationHosts: [ocsp.example.corp, crl.example.corp]
```

You may also configure your preferred certificate revocation checking mechanism with:

* `preferredCertRevocationChecker`

Use `-full` for detailed signer, certificate, timestamp and revocation output.

## PAdES Level

For ETSI.CAdES.detached signatures, pdfcpu applies supported Baseline B checks and reports `PAdES: B-B` when those checks and the local signature assessment succeed.

Timestamp, DSS, CRL and OCSP information is reported separately as available evidence.
Its presence does not promote the reported PAdES level beyond B-B.

The PAdES baseline levels are defined in [ETSI EN 319 142-1 V1.2.1 (2024-01)](https://www.etsi.org/deliver/etsi_en/319100_319199/31914201/01.02.01_60/en_31914201v010201p.pdf) 6.1.

| PAdES level | description | pdfcpu handling |
|:------------|:------------|:-------------|
| B-B | Basic electronic signature | supported profile result |
| B-T | B-B with validated signature timestamp | embedded token reported, but not authenticated or classified |
| B-LT | B-T with validation material | not classified or validated |
| B-LTA | B-LT with archive timestamp evidence | not classified or validated |

## Limitations

Current limitations mostly involve older cryptographic standards restricted by the Go runtime, missing checks for
permission violations after successful signature validation, incomplete embedded-signature-timestamp authentication,
and trust evidence that is not evaluated under a maintained external policy.

* Permissions handling:
  * DocMDP: missing document checks for permissions levels 2 and 3.
  * FieldMDP: not yet processed.
  * UR3: missing document checks for permissions defined by the UR transform method in the UR3 reference dictionary.
* Catalog DSS: missing processing of the VRI structure.
* Embedded signature timestamps: RFC 3161 tokens are located and parsed, but their cryptographic signature, message
  imprint, profile, and TSA certificate path are not yet authenticated.
* Document timestamps: supported RFC 3161 checks are performed locally, but pdfcpu does not establish historical trust
  at the token's generation time or apply a maintained trusted-list, qualified-TSA, legal, or enterprise policy.
* LTV: pdfcpu does not process VRI entries or classify and validate PAdES-B-T, B-LT, or B-LTA.
* Elliptic curve encryption algorithms: support needs to be extended as standards keep evolving.
* Legacy signatures: `adbe.x509.rsa_sha1` and `adbe.pkcs7.sha1` use SHA-1 and are supported only for validating existing PDFs. They are deprecated in PDF 2.0 and must not be used for new signatures.
* Go runtime restrictions: certificate chains using SHA-1 signatures may be rejected by the Go runtime.

## Examples

The following commands use signature fixtures from the pdfcpu test corpus.
Results depend on the current time, configured local certificate store, available revocation services and build configuration.

### Baseline B Profile

The certificate in `testPAdES_BB.pdf` has expired, so current validation reports an unknown result rather than presenting stale success output:

```text
$ pdfcpu signatures validate testPAdES_BB.pdf

1 form signature (authoritative, visible, signed) on page 1
  Integrity: signature authenticated, signed content digest verified
     Status: validity of the signature is unknown
     Reason: signer's certificate or one of its parent certificates has expired
     Signed: 2024-03-04 14:25:54 +0200
```

Use `--full` to inspect the Baseline B profile, certificate dates, local path and revocation evidence.
`PAdES: B-B` is only presented when the supported Baseline B checks and local signature assessment succeed.

### Embedded Timestamp Evidence

`testPAdES_BT.pdf` contains an embedded timestamp token.
The detailed output separates the authenticated document signature from the observed, but not yet authenticated,
signature timestamp:

```text
$ pdfcpu signatures validate --all --full testPAdES_BT.pdf

1:
form signature (authoritative, visible, signed) on page 1

Evidence:
  Cryptographic signature: authenticated
  Signed content digest:   verified
  Signature profile:       validated
  Signer certificate:      identified
  Certificate path:        unknown
  Revocation status:       unknown
  Timestamp:               signature timestamp observed at 2024-03-04 12:25:32 +0000

  Timestamp evidence:
    Cryptographic signature: unknown
    Message imprint:         unknown
    Profile:                 unknown
    Certificate path:        unknown

Assessment:
  Status:            validity of the signature is unknown
  Reason:            signer's certificate or one of its parent certificates has expired
  Signed:            2024-03-04 14:25:31 +0200
  Document modified: false
```

Timestamp presence does not authenticate the token or establish PAdES-B-T.

### Document Timestamp

`testPAdES_BLTA.pdf` contains a document timestamp and a form signature.
The document timestamp receives the supported RFC 3161 checks. In this example, its signature, message imprint, and
profile validate, while the configured local certificate store cannot resolve the TSA certificate path:

```text
$ pdfcpu signatures validate --all --full testPAdES_BLTA.pdf

1:
document timestamp (not locally validated, invisible, signed)

Evidence:
  Cryptographic signature: authenticated
  Signed content digest:   verified
  Signature profile:       validated
  Signer certificate:      identified
  Certificate path:        unknown
  Revocation status:       unknown
  Timestamp:               document timestamp observed at 2024-03-04 12:24:33 +0000

  Timestamp evidence:
    Cryptographic signature: authenticated
    Message imprint:         verified
    Profile:                 validated
    Certificate path:        unknown

Assessment:
  Status:            validity of the signature is unknown
  Reason:            signer's certificate path was not resolved using the configured local certificate store
  Signed:            2024-03-04 14:24:32 +0200
  Document modified: false
```

If all supported cryptographic and best-effort local checks succeed, the type is shown as `locally validated`.
This means locally validated within pdfcpu's technical scope. It does not establish historical trust at the generation
time or a policy-based, qualified-TSA, or legal-validity conclusion.

### Usage Rights Signature

`usageRights.pdf` contains a usage rights signature.
Its detailed output distinguishes the unresolved local certificate path from the signature evidence:

```text
$ pdfcpu signatures validate --full usageRights.pdf

1:
usage rights signature (invisible, signed)

Evidence:
  Cryptographic signature: authenticated
  Signed content digest:   verified
  Signature profile:       validated
  Signer certificate:      identified
  Certificate path:        unknown
  Revocation status:       unknown
  Timestamp:               not observed

Assessment:
  Status:            validity of the signature is unknown
  Reason:            signer's certificate path was not resolved using the configured local certificate store
  Signed:            2022-12-15 12:08:57 -0500
  Document modified: false
```

Importing an accepted missing issuer or root certificate with [pdfcpu certificates import](/core/certs_import) may allow pdfcpu to resolve the local certificate path.
The command checks the usage rights signature itself; violations of permissions defined by UR3 transform parameters are not currently checked.

Validate a signed PDF streamed from S3:

```sh
$ aws s3 cp s3://acme-signing/executed.pdf - \
   | pdfcpu signatures validate -
```
