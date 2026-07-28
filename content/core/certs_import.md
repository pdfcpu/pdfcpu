---
layout: default
title: "Import Certificates"
---

# Import Certificates

Import root or intermediate certificate material into pdfcpu's local certificate store.

Imported certificates are stored below `.../pdfcpu/certs` and are used by subsequent signature checks for local chain building.
If an imported file resolves to the same destination name as an installed file, pdfcpu replaces the installed file
transactionally. A failed multi-file import restores any files already replaced by that import.

```
pdfcpu certificates import inFile...
```

## Arguments

| name   | description                            | required |
|:-------|:---------------------------------------|:---------|
| inFile | certificate files: `.pem`, `.p7c`, `.cer`, `.crt` | yes |

## Example

If a signature chain cannot be checked because a trusted root or intermediate certificate is missing, import the missing certificate material into pdfcpu.

<p align="center">
  <img style="border-color:silver" border="1" src="../resources/certs1.png" height="100">
</p>

```sh
$ pdfcpu certificates import hr.p7c
hr.p7c: 156 certificates
imported 156 certificates
```

<p align="center">
  <img style="border-color:silver" border="1" src="../resources/certs2.png" height="120">
</p>

Importing certificates also parses them, meaning pdfcpu ensures it can handle the certificate file and any involved certificate algorithms.
This is important because these evolve over time and corresponding support will need to be implemented after the fact.

Popular PDF viewers can export their root CAs, but you have to make sure you are not violating any usage restrictions before importing them into pdfcpu.

Once your certificates are imported, you are free to move them around within `pdfcpu/certs`, including creating special folders.
