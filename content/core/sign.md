---
layout: default
title: "Signatures"
---

# Signatures

Signature commands inspect or remove digital signatures present in a PDF.

pdfcpu's open source signature handling focuses on PDF signature integrity:

* signed byte ranges
* CMS/PKCS#7 processing
* signer and certificate extraction
* best-effort checks against the configured local certificate store and available revocation information

It reports trust-related evidence such as certificate chains, timestamps, revocation responses, DSS data, and PAdES baseline indicators where available.

This is not a legal-validity, eIDAS, enterprise policy, or full long-term validation statement.

## Usage

```
pdfcpu signatures validate inFile [flags]
pdfcpu signatures remove inFile [ outFile ] [flags]
```

### [Common Flags](/getting_started/common_flags)
