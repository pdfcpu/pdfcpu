---
layout: default
title: "Signatures"
---

# Signatures

Signature commands inspect or remove digital signatures present in a PDF.

pdfcpu validates signature integrity, reports available trust evidence and performs a best-effort local assessment.

Signature validation is under active development. The current implementation focuses on:

* signed byte ranges
* CMS/PKCS#7 processing
* signer and certificate extraction
* best-effort checks against the configured local certificate store and available revocation information

This is not a legal-validity, eIDAS, enterprise policy, or full long-term validation statement.

## Usage

```
pdfcpu signatures validate inFile [flags]
pdfcpu signatures remove inFile [ outFile ] [flags]
```

### [Common Flags](/getting_started/common_flags)

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=48b1f0fe-bc76-4fa0-912a-dd771c5ca918" width="1" height="1" />
