---
layout: default
title: "Inspect Certificates"
---

# Inspect Certificates

Inspect certificate files before importing them into pdfcpu.

```
pdfcpu certificates inspect inFile...
```

## Arguments

| name   | description                            | required |
|:-------|:---------------------------------------|:---------|
| inFile | certificate files: `.pem`, `.p7c`, `.cer`, `.crt` | yes |

## Example

```sh
$ pdfcpu certificates inspect root.crt
1:
    Subject:
             org       : A-Trust GmbH
             unit      : a-sign-premium-mobile-seal-09
             name      : a-sign-premium-mobile-seal-09
             country   : AT
     Issuer:
             org       : A-Trust GmbH
             unit      : A-Trust-Root-09
             name      : A-Trust-Root-09
             country   : AT
       from: 2023-02-21
       thru: 2036-07-14
         CA: true

inspected 1 certificates
```
