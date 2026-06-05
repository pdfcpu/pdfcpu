---
layout: default
title: "Validate Signatures"
---

# Validate Signatures

Validate signature integrity for digital signatures present in a PDF.

This command checks whether signed byte ranges still match their signatures and reports available signer, certificate, timestamp, revocation, DSS, and PAdES evidence.

Trust-related output is based on pdfcpu's configured local certificate store and available revocation information.
It is useful for inspection and automation, but it is not a legal-validity, eIDAS, enterprise policy, or full long-term validation statement.

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

These checks are useful for inspection and automation, but they are not a substitute for a dedicated trust policy, compliance profile, or legal-validity assessment.

## Checking Revocation

Certificates may be revoked for various reasons.
Checking the revocation status may require online access and depends on the certificate, the responder, and any embedded evidence in the PDF.

You can configure timeout values for CRL and OCSP responders with:

* `timeoutCRL`
* `timeoutOCSP`

You may also configure your preferred certificate revocation checking mechanism with:

* `preferredCertRevocationChecker`

Use `-full` for detailed signer, certificate, timestamp and revocation output.

## PAdES Level

While the PDF specification mainly focuses on PAdES-E-BES and PAdES-E-EPES for processing ETSI.CAdES.detached signatures, pdfcpu detects and reports PAdES Baseline evidence:

* B-B
* B-T
* B-LT
* B-LTA

PAdES-B levels are widely adopted and useful for describing the evidence present in modern signed PDFs.
In the open source build, higher levels are reported as evidence indicators.
They should not be read as a full long-term validation or compliance result.

The PAdES baseline levels are defined in [ETSI EN 319 142-1 V1.2.1 (2024-01)](https://www.etsi.org/deliver/etsi_en/319100_319199/31914201/01.02.01_60/en_31914201v010201p.pdf) 6.1.

| PAdES level | description | OSS handling |
|:------------|:------------|:-------------|
| B-B | Basic electronic signature | validated/reported |
| B-T | B-B with timestamp evidence or DTS | detected/reported |
| B-LT | B-T with embedded CRL and OCSP data | evidence detected/reported |
| B-LTA | B-LT with archive timestamp evidence | evidence detected/reported |

pdfcpu OSS currently focuses on signature integrity and PAdES-B-B validation.
Full trust-policy and LTV validation belong to a dedicated trust validation layer.

## Limitations

Current limitations mostly involve either older encryption standards restricted by the Go runtime for security reasons, missing checks for permission violations after successful signature validation, or trust evidence that is detected but not fully policy-validated.

* Permissions handling:
  * DocMDP: missing document checks for permissions levels 2 and 3.
  * FieldMDP: not yet processed.
  * UR3: missing document checks for permissions defined by the UR transform method in the UR3 reference dictionary.
* Catalog DSS: missing processing of the VRI structure.
* Timestamps and LTV: timestamp, DSS, CRL and OCSP evidence may be detected and reported, but pdfcpu OSS does not perform full RFC3161 trust validation, VRI processing, PAdES-B-LT/B-LTA validation, or enterprise policy validation.
* Elliptic curve encryption algorithms: support needs to be extended as standards keep evolving.
* Go runtime restrictions: no support for SHA-1, which is considered insecure.

## Examples

We start with an ETSI CAdES-detached signature for which pdfcpu reports valid signature integrity and PAdES-B-B evidence:

```sh
$ pdfcpu signatures validate sample1.pdf
optimizing...

1 form signature (authoritative, visible, signed) on page 1
   Status: signature is valid
   Reason: document has not been modified
   Signed: 2025-03-18 10:07:18 +0000
```

By using `-full` we can look at all the details:

```sh
$ pdfcpu signatures validate sample1.pdf --full
optimizing...

1:
       Type: form signature (authoritative, visible, signed) on page 1
     Status: signature is valid
     Reason: document has not been modified
     Signed: 2025-03-18 10:07:18 +0000
DocModified: false
    Details:
             SubFilter:      ETSI.CAdES.detached
             SignerIdentity: John Doe
             SignerName:     John Doe
             SigningTime:    2025-03-18 10:07:18 +0000
     Signer:
             Timestamp:      false
             LTVEnabled:     false
             PAdES:          B-B
             Certified:      false
             Authoritative:  true
             Certificate:
                             Subject:    John Doe
                             Issuer:     a-sign-premium-mobile-05
                             Expired:    false
                             Qualified:  true
                             Trust:      Status: ok
                                        Reason: cert chain resolved by local trust store
                             Revocation: Status: ok
                                         Reason: not revoked (CRL check ok)
```

We can see the reported PAdES evidence and the certificate chain resolved against pdfcpu's local certificate store.
The output also shows that the certificate is not expired and that the available CRL check returned ok.

Next we take a look at a signature that, in addition to PAdES B-B evidence, also contains timestamp evidence.

```sh
$ pdfcpu signatures validate sample2.pdf
optimizing...

1 form signature (authoritative, visible, signed) on page 1
   Status: signature is valid
   Reason: document has not been modified
   Signed: 2024-09-19 13:09:06 +0000
```

In addition to `-full` we are also going to supply `-all` to check for other signatures:

```sh
$ pdfcpu signatures validate sample2.pdf --all --full
repaired: trailer size
optimizing...

1:
       Type: form signature (authoritative, visible, signed) on page 1
     Status: signature is valid
     Reason: document has not been modified
     Signed: 2024-09-19 13:09:06 +0000
    Details:
             SubFilter:      ETSI.CAdES.detached
             SignerIdentity: John Doe
             SigningTime:    2024-09-19 13:09:06 +0000
     Signer:
             Timestamp:      2024-09-19 13:10:03 +0000
             LTVEnabled:     false
             PAdES:          B-T
             Certified:      false
             Authoritative:  true
```

Using `-all` reveals that there is only one signature.
The signature contains a single signer, which is the expected behavior for ETSI CAdES-detached signatures.

We see the certificate chain resolved against pdfcpu's local certificate store, and also that the certificate is not expired and the available OCSP responder returned a non-revoked status.
We can see timestamp evidence, which pdfcpu reports as B-T evidence.
This could also be due to a separate DTS.

Next, we have an example that contains a Document Timestamp Signature as timestamp evidence.

```sh
$ pdfcpu signatures validate sample3.pdf
optimizing...

2 signatures present:

1:
     Type: document timestamp (trusted, invisible, signed)
   Status: signature is valid
   Reason: document has not been modified
   Signed: 2024-03-04 12:24:33 +0000

2:
     Type: form signature (authoritative, visible, signed) on page 1
   Status: signature is valid
   Reason: document has not been modified
   Signed: 2024-03-04 12:24:31 +0000
```

In order to see the details for both signatures, supply `--all` and `--full`.
Using a combination of short flags also works: `-af`.
The `trusted` label shown for the document timestamp is pdfcpu's current CLI output for a successfully processed DTS and should be read in the context of the OSS limitations described above.

At last, we take a look at a PDF with a usage rights signature.
This is not a signature in the traditional sense, but rather a signed definition of permissions that PDF processors should obey.

```sh
$ pdfcpu signatures validate usageRights.pdf --all
optimizing...

1 usage rights signature (invisible, signed)
   Status: validity of the signature is unknown
   Reason: signers certificate chain is not in the configured local trusted certificate store
   Signed: 2022-12-15 17:08:57 +0000
```

Using `-full` explains what is going on:

```sh
$ pdfcpu signatures validate usageRights.pdf --full
optimizing...

1:
       Type: usage rights signature (invisible, signed)
     Status: validity of the signature is unknown
     Reason: signers certificate chain is not in the configured local trusted certificate store
     Signed: 2022-12-15 17:08:57 +0000
DocModified: false
    Details:
             SubFilter:      adbe.pkcs7.detached
             SignerName:     ARE Production V8.1 G3 P24 1007685
     Signer:
             Timestamp:      false
             LTVEnabled:     false
             Certified:      false
             Authoritative:  false
             Certificate:
                             Subject:    ARE Production V8.1 G3 P24 1007685
                             Issuer:     Adobe Product Services G3
                             Expired:    false
                             Trust:      Status: not ok
                                         Reason: certificate not trusted
                             Revocation: Status: ok
                                         Reason: not revoked (CRL check ok)
             Problems:       certificate verification failed: x509: certificate signed by unknown authority
```

A problem points to missing intermediate or root certificates.
The certificate is therefore not trusted.

Conclusion: If you import the missing certificates using [pdfcpu certificates import](/core/certs_import), pdfcpu should be able to complete its local certificate-chain check for this usage rights signature.

This command only checks the usage rights signature itself.
Any violation of usage rights defined as UR3 transform parameters are not checked at the moment.

Validate a signed PDF streamed from S3:

```sh
$ aws s3 cp s3://acme-signing/executed.pdf - \
   | pdfcpu signatures validate -
```
