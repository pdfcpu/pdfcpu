---
layout: default
title: "Signatures"
---

# Signatures

Signature commands inspect or remove digital signatures present in a PDF.

pdfcpu validates signature integrity, reports available trust evidence and performs a best-effort local assessment.

The cryptographic evidence and the broader trust assessment are reported separately. A signature may authenticate and
its signed-content digest may verify even when the configured local certificate store cannot establish a certificate
path or revocation status.

Signature validation is under active development.<br>
The current implementation focuses on:

* signed byte ranges
* CMS/PKCS#7 processing
* signer and certificate extraction
* local RFC 3161 document-timestamp validation
* detection and reporting of embedded signature timestamp tokens
* best-effort checks against the configured local certificate store and available revocation information

Embedded signature timestamp tokens are not yet authenticated.<br>
This is a current implementation gap in the validation
coverage.

The reported assessment is not a legal-validity, eIDAS, qualified-signature, enterprise-policy, or full
long-term-validation statement.

## Usage

```
pdfcpu signatures validate inFile [flags]
pdfcpu signatures remove inFile [ outFile ] [flags]
```

### [Common Flags](/getting_started/common_flags)

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=48b1f0fe-bc76-4fa0-912a-dd771c5ca918" width="1" height="1" />
