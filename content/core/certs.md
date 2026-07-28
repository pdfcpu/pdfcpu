---
layout: default
title: "Certificates"
---

# Certificates

Certificate commands manage pdfcpu's local trusted certificate store.

pdfcpu uses this store for local certificate-chain checks during signature validation.
Standard builds start with an empty trusted certificate store.
Builds created with `-tags pdfcpu_eutl` initialize this store with an embedded snapshot of EU Trusted List certificate bundles.

Importing a certificate does not by itself establish legal validity, eIDAS compliance, or an enterprise trust policy.
It only makes certificate material available to pdfcpu for local trust-chain inspection.

## Usage

```
pdfcpu certificates list
pdfcpu certificates list --json
pdfcpu certificates inspect inFile
pdfcpu certificates import inFile...
pdfcpu certificates reset
```

### [Common Flags](/getting_started/common_flags)
