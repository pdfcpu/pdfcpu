---
layout: default
title: "Validate Signatures"
---

# Validate Signatures

Validate signature integrity, report available trust evidence and perform a best-effort local assessment.

This command checks whether signed byte ranges still match their signatures and reports available signer, certificate, timestamp, revocation, DSS, and PAdES evidence.

Certificate-path and revocation output is based on pdfcpu's configured local certificate store and available revocation information.
It is useful for inspection and automation, but it is not a legal-validity, eIDAS, enterprise policy, or full long-term validation statement.

Signature validation is under active development.
Support for additional signature formats, validation rules, and trust evidence is being expanded.

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
| B-T | B-B with validated signature timestamp | not classified or validated |
| B-LT | B-T with validation material | not classified or validated |
| B-LTA | B-LT with archive timestamp evidence | not classified or validated |

## Limitations

Current limitations mostly involve either older cryptographic standards restricted by the Go runtime for security reasons, missing checks for permission violations after successful signature validation, or trust evidence that is detected but not fully policy-validated.

* Permissions handling:
  * DocMDP: missing document checks for permissions levels 2 and 3.
  * FieldMDP: not yet processed.
  * UR3: missing document checks for permissions defined by the UR transform method in the UR3 reference dictionary.
* Catalog DSS: missing processing of the VRI structure.
* Timestamps and LTV: timestamp, DSS, CRL and OCSP evidence may be detected and reported, but pdfcpu does not perform full RFC3161 trust validation, VRI processing, PAdES-B-T/B-LT/B-LTA classification or validation, or enterprise policy validation.
* Elliptic curve encryption algorithms: support needs to be extended as standards keep evolving.
* Legacy signatures: `adbe.x509.rsa_sha1` and `adbe.pkcs7.sha1` use SHA-1 and are supported only for validating existing PDFs. They are deprecated in PDF 2.0 and must not be used for new signatures.
* Go runtime restrictions: certificate chains using SHA-1 signatures may be rejected by the Go runtime.

## Examples

The following commands use fixtures from `pkg/samples/signatures`.
Results depend on the current time, configured local certificate store, available revocation services and build configuration.

### Baseline B Profile

The certificate in `testPAdES_BB.pdf` has expired, so current validation reports an unknown result rather than presenting stale success output:

```text
$ pdfcpu signatures validate pkg/samples/signatures/ETSI.CAdES.detached/testPAdES_BB.pdf
optimizing...

1 form signature (authoritative, visible, signed) on page 1
   Status: validity of the signature is unknown
   Reason: signer's certificate or one of its parent certificates has expired
   Signed: 2024-03-04 14:25:54 +0200
```

Use `--full` to inspect the Baseline B profile, certificate dates, local path and revocation evidence.
`PAdES: B-B` is only presented when the supported Baseline B checks and local signature assessment succeed.

### Embedded Timestamp Evidence

`testPAdES_BT.pdf` contains an embedded timestamp token.
The detailed output reports the observed time separately and explains the current assessment:

```text
$ pdfcpu signatures validate pkg/samples/signatures/ETSI.CAdES.detached/testPAdES_BT.pdf -a -f
optimizing...

1:
       Type: form signature (authoritative, visible, signed) on page 1
     Status: validity of the signature is unknown
     Reason: signer's certificate or one of its parent certificates has expired
     Signed: 2024-03-04 14:25:31 +0200
DocModified: unknown
    Details:
             SubFilter:      ETSI.CAdES.detached
             SignerIdentity: Unknown
             SignerName:
             ContactInfo:
             Location:
             Reason:
             SigningTime:    2024-03-04 14:25:31 +0200
             Field:          SignatureFieldName 25
     Signer:
             Timestamp:      2024-03-04 12:25:32 +0000
             Certified:      false
             Authoritative:  true
             Certificate:
                             Subject:    TEST Testovyi Test
                             Issuer:     Administrator ITS CCA (CA TEST)
                             SerialNr:   233277b9179888b4040000000b080000fd780000
                             Valid From: 2024-03-03 22:00:00 +0000
                             Valid Thru: 2026-03-03 21:59:59 +0000
                             Expired:    true
                             QC Policy:  false
                             CA:         false
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   2048 bits
                             SelfSigned: false
                             Local Path: Status: unknown
                                         Reason: certificate path was not resolved using the configured local
                                                 certificate store
                             Revocation: Local:  unknown
             Problems:       pkcs7: embedded timestamp token observed but not fully authenticated
                             certificate verification failed for serial="233277b9179888b4040000000b080000fd780000":
                             pkcs7: verify certificate chain: x509: certificate has expired or is not yet valid: current
                             time 2026-08-10T00:48:58+02:00 is after 2026-03-03T21:59:59Z
```

Timestamp presence does not establish or report PAdES-B-T.

### Document Timestamp

`testPAdES_BLTA.pdf` contains a document timestamp and a form signature.
When the TSA certificate path cannot be resolved using the configured local certificate store, the summary says so directly:

```text
$ pdfcpu signatures validate pkg/samples/signatures/ETSI.CAdES.detached/testPAdES_BLTA.pdf --all
optimizing...

2 signatures present:
1 signed form signature (1 visible)
1 signed doc timestamp signature (0 visible)

1:
     Type: document timestamp (not locally validated, invisible, signed)
   Status: validity of the signature is unknown
   Reason: signer's certificate path was not resolved using the configured local certificate store

2:
     Type: form signature (authoritative, visible, signed) on page 1
   Status: validity of the signature is unknown
   Reason: signer's certificate or one of its parent certificates has expired
```

If the supported cryptographic and best-effort local checks succeed, the type is shown as `locally validated`.
This remains a local assessment, not a policy-based trust or legal-validity conclusion.

### Usage Rights Signature

`usageRights.pdf` contains a usage rights signature.
Its detailed output distinguishes the unresolved local certificate path from the signature evidence:

```text
$ pdfcpu signatures validate pkg/samples/signatures/adbe.pkcs7.detached/usageRights.pdf -f
optimizing...

1:
       Type: usage rights signature (invisible, signed)
     Status: validity of the signature is unknown
     Reason: signer's certificate path was not resolved using the configured local certificate store
     Signed: 2022-12-15 12:08:57 -0500
DocModified: unknown
    Details:
             SubFilter:      adbe.pkcs7.detached
             SignerIdentity: Unknown
             SignerName:     ARE Production V8.1 G3 P24 1007685
             ContactInfo:
             Location:
             Reason:
             SigningTime:    2022-12-15 12:08:57 -0500
             Field:
     Signer:
             Timestamp:      false
             Certified:      false
             Authoritative:  false
             Certificate:
                             Subject:    ARE Production V8.1 G3 P24 1007685
                             Issuer:     Adobe Product Services G3
                             SerialNr:   901357a46c30d17b2f7d64b453c0818
                             Valid From: 2022-02-11 00:00:00 +0000
                             Valid Thru: 2035-12-31 23:59:59 +0000
                             Expired:    false
                             QC Policy:  false
                             CA:         false
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   2048 bits
                             SelfSigned: false
                             Local Path: Status: unknown
                                         Reason: certificate path was not resolved using the configured local
                                                 certificate store
                             Revocation: Local:  unknown
                                         Reason: OCSP: no conclusive current response: responder 1
                                                 (http://pki-ocsp.symauth.com): OCSP: send request to
                                                 http://pki-ocsp.symauth.com: Post "http://pki-ocsp.symauth.com": lookup
                                                 pki-ocsp.symauth.com: no such host
                                                 CRL: fetch
                                                 http://pki-crl.symauth.com/ca_7a5c3a0c73117406add19312bc1bc23f/
                                                 LatestCRL.crl: Get
                                                 "http://pki-crl.symauth.com/ca_7a5c3a0c73117406add19312bc1bc23f/
                                                 LatestCRL.crl": lookup pki-crl.symauth.com: no such host
             IntermediateCA:
                             Subject:    Adobe Product Services G3
                             Issuer:     Adobe Root CA G2
                             SerialNr:   ca8b6547b89e6d2068975cd8b9b89e2
                             Valid From: 2016-11-29 00:00:00 +0000
                             Valid Thru: 2041-11-28 23:59:59 +0000
                             Expired:    false
                             QC Policy:  false
                             CA:         true
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   4096 bits
                             SelfSigned: false
                             Local Path: Status: unknown
                                         Reason: certificate path was not resolved using the configured local
                                                 certificate store
             RootCA:
                             Subject:    Adobe Root CA G2
                             Issuer:     Adobe Root CA G2
                             SerialNr:   5df12f5f57a7c3e1b002d893270cdde1
                             Valid From: 2016-11-29 00:00:00 +0000
                             Valid Thru: 2046-11-28 23:59:59 +0000
                             Expired:    false
                             QC Policy:  false
                             CA:         true
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   4096 bits
                             SelfSigned: true
                             Local Path: Status: unknown
                                         Reason: certificate path was not resolved using the configured local
                                                 certificate store
             Problems:       certificate path was not resolved using the configured local certificate store for
                             serial="901357a46c30d17b2f7d64b453c0818": pkcs7: verify certificate chain: x509:
                             certificate signed by unknown authority
                             import missing certificates into pdfcpu's local certificate store with "pdfcpu certificates
                             import <file>"
                             certificate revocation check for serial="901357a46c30d17b2f7d64b453c0818" using CRL: CRL:
                             fetch http://pki-crl.symauth.com/ca_7a5c3a0c73117406add19312bc1bc23f/LatestCRL.crl: Get
                             "http://pki-crl.symauth.com/ca_7a5c3a0c73117406add19312bc1bc23f/LatestCRL.crl": lookup
                             pki-crl.symauth.com: no such host
                             certificate revocation check for serial="901357a46c30d17b2f7d64b453c0818": OCSP: no
                             conclusive current response: responder 1 (http://pki-ocsp.symauth.com): OCSP: send request
                             to http://pki-ocsp.symauth.com: Post "http://pki-ocsp.symauth.com": lookup
                             pki-ocsp.symauth.com: no such host
                             CRL: fetch http://pki-crl.symauth.com/ca_7a5c3a0c73117406add19312bc1bc23f/LatestCRL.crl:
                             Get "http://pki-crl.symauth.com/ca_7a5c3a0c73117406add19312bc1bc23f/LatestCRL.crl": lookup
                             pki-crl.symauth.com: no such host
```

Importing an accepted missing issuer or root certificate with [pdfcpu certificates import](/core/certs_import) may allow pdfcpu to resolve the local certificate path.
The command checks the usage rights signature itself; violations of permissions defined by UR3 transform parameters are not currently checked.

Validate a signed PDF streamed from S3:

```sh
$ aws s3 cp s3://acme-signing/executed.pdf - \
   | pdfcpu signatures validate -
```
