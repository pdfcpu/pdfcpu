---
layout: default
title: "Reset Certificates"
---

# Reset Certificates

Reset certificates managed by pdfcpu to the build defaults.

```
pdfcpu certificates reset
```

Standard builds reset to an empty trusted certificate store.
Builds created with `-tags pdfcpu_eutl` reset to the embedded snapshot of EU Trusted List certificate bundles.

## Example

```sh
$ pdfcpu certificates reset
Are you ready to reset your trusted certificates to the build defaults?
(yes/no): yes
resetting..
Finished
```
