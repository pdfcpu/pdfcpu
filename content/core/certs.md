---
layout: default
title: "Certificates"
---

# Certificates



* Have a look at some [examples](#examples).


## Usage

```
pdfcpu certificates list
pdfcpu certificates inspect inFile
pdfcpu certificates import inFile..
pdfcpu certificates reset
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description                | required 
|:-------------|:---------------------------|:--------
| inFile       | certificate(s) .pem, .p7c, .cer, .crt|   yes

<br>

## Examples

pdfcpu uses a certificate store located in the pdfcpu configuration directory.
Standard builds start with an empty trusted certificate store.
Builds created with `-tags pdfcpu_eutl` initialize this store with an embedded snapshot of EU Trusted List certificate bundles.
This makes it explicit which certificates are used for local signature checks.

The certificate store is used for pdfcpu's local certificate-chain checks.
Importing a certificate does not by itself establish legal validity, eIDAS compliance, or an enterprise trust policy.

Use `pdfcpu certificates list` to inspect the certificates currently managed by pdfcpu:

```sh
$ pdfcpu certificates list
certDir: /Users/horstrutter/Library/Application Support/pdfcpu/certs

...
```

<br>

If a signature chain cannot be checked because a trusted root or intermediate certificate is missing, import the missing certificate material into pdfcpu.
Imported certificates are stored below `.../pdfcpu/certs` and are used by subsequent signature checks for local chain building.

<p align="center">
  <img style="border-color:silver" border="1" src="../resources/certs1.png" height="100">
</p>

The recommended way to achieve this is:

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
<br><br>
Case in point - the elliptic curve algorithms which are constantly improved.


**Hint:** 
Popular PDF Viewers can export their rootCAs, but you have to make sure you are not violating any usage restrictions before importing them into pdfcpu.

Once your certs are imported you are free to move them around within `pdfcpu/certs` any way you like including
creating special folders.

<br>

If you want to reset certificates managed by pdfcpu do this:

```sh
$ pdfcpu certificates reset
Are you ready to reset your trusted certificates to the build defaults?
(yes/no): yes
resetting..
Finished
```

<br>

You may also inspect your certificate file(s) before importing:

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

