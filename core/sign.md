---
layout: default
---

# Signatures


Usage Rights Signature

 to detect changes to a document that
shall invalidate a usage rights signature,

* Have a look at some [examples](#examples).


## Usage

```
pdfcpu signatures validate [-a(ll) -f(ull)] -- inFile
```

<br>

### Flags

| name   | description             | default | required
|:-------|:------------------------|-------------------
| a(ll)  | validate all signatures | false   |no
| f(ull) | comprehensive output    | false   |no

<br>


### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

<br>

### Arguments

| name         | description         | required 
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

unsupported:
* limited support for DocMDP entry
* limited support for DSS entry
* limited support for UR
* archive timestamps

restrictions imposed by Go ecosystem:
* no support for SHA-1 as considered unsafe

supported PDF signature dict subfilters:
All subfilters as defined by the PDF specification are supported.
Beware you will likely hit roadblocks when trying to validate
signatures of type `adbe.x509.rsa.sha1` and `adbe.pkcs7.sha1` since SHA-1 is considered unsafe by the Go runtime.

explain when using all may make sense in the context of authorative and certified

ETSI-sample without timestamp compact vs. full

ETSI-sample with timestamp B-B compact vs. full

B-LTA sample with DTS compact vs. full

usage rights sample compact vs. full



```sh
$ pdfcpu sig val -all usageRights.pdf
optimizing...

1 usage rights signature (invisible, signed)
   Status: validity of the signature is unknown
   Reason: signer's certificate chain is not in the trusted list of Root CAs
   Signed: 2022-12-15 17:08:57 +0000
```

<br>

Using `-all` reveals there is only one signature.
Let's take a detailed look at what is going on here:

```sh
$ pdfcpu sig val -full usageRights.pdf
optimizing...

1:
       Type: usage rights signature (invisible, signed)
     Status: validity of the signature is unknown
     Reason: signer's certificate chain is not in the trusted list of Root CAs
     Signed: 2022-12-15 17:08:57 +0000
DocModified: false
    Details:
             SubFilter:      adbe.pkcs7.detached
             SignerIdentity: Unknown
             SignerName:     ARE Production V8.1 G3 P24 1007685
             ContactInfo:
             Location:
             Reason:
             SigningTime:    2022-12-15 17:08:57 +0000
             Field:
     Signer:
             Timestamp:      false
             LTVEnabled:     false
             Certified:      false
             Authoritative:  false
             Certificate:
                             Subject:    ARE Production V8.1 G3 P24 1007685
                             Issuer:     Adobe Product Services G3
                             SerialNr:   901357a46c30d17b2f7d64b453c0818
                             Valid From: 2022-02-11 00:00:00 +0000
                             Valid Thru: 2035-12-31 23:59:59 +0000
                             Expired:    false
                             Qualified:  false
                             CA:         false
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   2048 bits
                             SelfSigned: false
                             Trust:      Status: not ok
                                         Reason: certificate not trusted
                             Revocation: Status: ok
                                         Reason: not revoked (CRL check ok)
             IntermediateCA:
                             Subject:    Adobe Product Services G3
                             Issuer:     Adobe Root CA G2
                             SerialNr:   ca8b6547b89e6d2068975cd8b9b89e2
                             Valid From: 2016-11-29 00:00:00 +0000
                             Valid Thru: 2041-11-28 23:59:59 +0000
                             Expired:    false
                             Qualified:  false
                             CA:         true
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   4096 bits
                             SelfSigned: false
                             Trust:      Status: ok
                                         Reason: CA
             RootCA:
                             Subject:    Adobe Root CA G2
                             Issuer:     Adobe Root CA G2
                             SerialNr:   5df12f5f57a7c3e1b002d893270cdde1
                             Valid From: 2016-11-29 00:00:00 +0000
                             Valid Thru: 2046-11-28 23:59:59 +0000
                             Expired:    false
                             Qualified:  false
                             CA:         true
                             Usage:
                             Version:    3
                             SignAlg:    RSA
                             Key Size:   4096 bits
                             SelfSigned: true
                             Trust:      Status: ok
                                         Reason: self signed
             Problems:       certificate verification failed for serial="901357a46c30d17b2f7d64b453c0818": x509: certificate signed by unknown authority
```