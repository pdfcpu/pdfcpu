---
layout: default
title: "Signatures"
---

# Signatures

Validate digital signatures present in a PDF.

pdfcpu's open source signature handling focuses on PDF signature integrity:
* signed byte ranges
* CMS/PKCS#7 processing
* signer and certificate extraction
* best-effort checks against the configured local certificate store and
available revocation information.

It reports trust-related evidence such as:
* certificate chains
* timestamps
* revocation responses
* DSS data
* PAdES baseline indicators

where available.

This is not a legal-validity, eIDAS, enterprise policy, or full long-term
validation (LTV) statement.

PDF supports several types of signatures, each with a distinct purpose:

### Form Signature
A digital signature associated with a form field within the document.  
It is primarily intended to authenticate the person who filled out the form and confirms the integrity of the entered data.  

### Page Signature
A digital signature applied directly onto a page, often as an annotation or widget.
Its purpose is to authenticate the visible content of the page, ensuring that it has not been altered.  

### Document Timestamp Signature (DTS)
A signature based on an [RFC 3161](https://datatracker.ietf.org/doc/html/rfc3161) `TimeStampToken`.
A DTS is evidence that a document existed at a specific point in time, without binding it to a particular signer.
Usually associated with PDFs prepared for long term validation.

### Usage Rights Signature
A special signature used to enable extended features (such as form filling, commenting, and saving) in PDF viewers like Adobe Reader.
It also detects unauthorized changes that would invalidate these usage rights.
Has to be the only signature in the document.

---

## Summary of Signature Intentions

| **Type**         | **Intention**                             | **Visibility**          |
|:----------------------------|:--------------------------------------------------|:-------------------------|
| **Form Signature**          | Authenticate form data and signer identity        | Visible or invisible     |
| **Page Signature**          | Authenticate page content and appearance          | Visible or invisible     |
| **Document Timestamp Signature** | Prove document existence at a point in time      | invisible         |
| **Usage Rights Signature**  | Define locked features, detect tampering     | invisible         |

<br>

> **Note:**  
> This is not intended as an in-depth introduction to PDF digital signatures.  
> For complete details, please refer to the [PDF 2.0 specification (ISO 32000-2:2020)](https://www.pdfa-inc.org/product/iso-32000-2-pdf-2-0-bundle-sponsored-access/).

<br>

> **Note:**  
> It may not be immediately obvious whether a PDF contains signatures.  
> You can check for existing signatures using `pdfcpu info` on the command line.

<br>

The validation steps are:

### 1. Check Hash of signed bytes
Compare the hash from the signature with a computed hash to detect any document modifications.

### 2. Verify Crypto Signature
Check that the signature was created using the correct private key and matches the data.

### 3. Check Certificate Evidence
Check the signer certificate and report whether it chains up to a certificate in pdfcpu's configured local certificate store.
pdfcpu also reports certificate validity dates and performs best-effort revocation checks when suitable CRL or OCSP information is available.

These checks are useful for inspection and automation, but they are not a substitute for a dedicated trust policy, compliance profile, or legal-validity assessment.
<br>

### Checking Revocation
Certificates may be revoked for various reasons.  
Checking the revocation status may require online access and depends on the certificate, the responder, and any embedded evidence in the PDF.

You can configure your timeout values for [CRL](https://en.wikipedia.org/wiki/Certificate_revocation_list) and [OCSP](https://en.wikipedia.org/wiki/Online_Certificate_Status_Protocol) responders with these configuration parameters:

- `timeoutCRL`
- `timeoutOCSP`

You may also configure your preferred certificate revocation checking mechanism (crl or ocsp) with the configuration parameter:

- `preferredCertRevocationChecker`

<br>

> **Note:**  
> Use `-full` for detailed signer, certificate, timestamp and revocation output.

<br>

Have a look at some [examples](#examples).

## Usage

```
pdfcpu signatures remove inFile [ outFile ] [flags]
pdfcpu signatures validate inFile [flags]
```

<br>

### Flags

| **Name** | **Description**          | **Default** | **Required** |
|:---------|:--------------------------|:------------|:-------------|
| rmenc    | Remove encryption while removing signatures | false | no |
| a(ll)    | Validate all signatures    | false       | no           |
| f(ull)   | Detailed output       | false       | no           |

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required 
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout for `remove` | no

<br>

### Certified and Authoritative Signatures

A **certified signature** is a special type of signature that locks the document at a certain point, allowing only certain permitted changes afterward. It indicates that the document was approved in its original form by the certifying party.

An **authoritative signature** is the first signature encountered in the document when no certified signature is present. pdfcpu uses it as the primary signature to inspect when no certified signature is present.

Any number of **approval signatures** may be applied after a certified signature.

By default, validation focuses only on the certified signature, if available, or otherwise the authoritative signature.

If the `-all` option is set, **all** signatures in the PDF are validated.

<br>

## Limitations

Current limitations mostly involve either older encryption standards restricted by the Go runtime for security reasons, missing checks for permission violations after successful signature validation, or trust evidence that is detected but not fully policy-validated.

- **Permissions Handling**:
  - **DocMDP**: Missing document checks for permissions levels 2 and 3.
  - **FieldMDP**: Not yet processed.
  - **UR3**: Missing document checks for permissions defined by the UR transform method in the UR3 reference dictionary.

- **Catalog DSS**: Missing processing of the VRI (Validation-Related Information) structure.

- **Timestamps and LTV**: Timestamp, DSS, CRL and OCSP evidence may be detected and reported, but pdfcpu OSS does not perform full RFC3161 trust validation, VRI processing, PAdES-B-LT/B-LTA validation, or enterprise policy validation.

- **Elliptic Curve Encryption Algorithms**: support needs to be extended as standards keep evolving.

- **Go Runtime Restrictions**: No support for SHA-1, which is considered insecure.

## PAdES Level

While the PDF specification mainly focuses on PAdES-E-BES and PAdES-E-EPES for processing ETSI.CAdES.detached signatures, pdfcpu detects and reports PAdES Baseline evidence:
* B-B
* B-T
* B-LT
* B-LTA

PAdES-B levels (Basic, Timestamp, Long-Term, Long-Term-Archival) are widely adopted and useful for describing the evidence present in modern signed PDFs.

In the open source build, higher levels are reported as evidence indicators. They should not be read as a full long-term validation or compliance result.

The PAdES baseline levels(profiles) are defined in [ETSI EN 319 142-1 V1.2.1 (2024-01)](https://www.etsi.org/deliver/etsi_en/319100_319199/31914201/01.02.01_60/en_31914201v010201p.pdf) 6.1.

| PAdES Level | Description                         | OSS handling |
|:------------|:------------------------------------|:-------------|
| B-B         | Basic electronic signature          | validated/reported |
| B-T         | B-B with timestamp evidence or DTS  | detected/reported |
| B-LT        | B-T with embedded CRL and OCSP data | evidence detected/reported |
| B-LTA       | B-LT with archive timestamp evidence | evidence detected/reported |

> **Note:**  
> pdfcpu OSS currently focuses on signature integrity and PAdES-B-B validation. Full trust-policy and LTV validation belong to a dedicated trust validation layer.

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

By using *-full* we can look at all the details: 

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
             ContactInfo:
             Location:       oesterreich.gv.at PDF Signatur
             Reason:         Signatur
             SigningTime:    2025-03-18 10:07:18 +0000
             Field:          Signature15430ca9-5df6-4b11-b423-ab48ec2439d6
     Signer:
             Timestamp:      false
             LTVEnabled:     false
             PAdES:          B-B
             Certified:      false
             Authoritative:  true
             Certificate:
                             Subject:    John Doe
                             Issuer:     a-sign-premium-mobile-05
                             SerialNr:   614a81f67
                             Valid From: 2023-01-04 10:39:36 +0000
                             Valid Thru: 2028-01-04 10:39:36 +0000
                             Expired:    false
                             Qualified:  true
                             CA:         false
                             Usage:
                             Version:    3
                             SignAlg:    ECDSA
                             Key Size:   256 bits
                             SelfSigned: false
                             Trust:      Status: ok
                                        Reason: cert chain resolved by local trust store
                             Revocation: Status: ok
                                         Reason: not revoked (CRL check ok)
             RootCA:
                             Subject:    a-sign-premium-mobile-05
                             Issuer:     A-Trust-Root-05
                             SerialNr:   36a009c2
                             Valid From: 2022-12-19 09:15:01 +0000
                             Valid Thru: 2029-07-10 07:15:01 +0000
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

```

We can see the reported PAdES evidence and the certificate chain resolved against pdfcpu's local certificate store.
The output also shows that the certificate is not expired and that the available CRL check returned ok.

<br>

Next we take a look at a signature that, in addition to PAdES B-B evidence, also contains timestamp evidence.

```sh
$ pdfcpu signatures validate sample2.pdf
optimizing...

1 form signature (authoritative, visible, signed) on page 1
   Status: signature is valid
   Reason: document has not been modified
   Signed: 2024-09-19 13:09:06 +0000
```

Now let's look at the validation result details.  
In addition to -full we are also going to supply -all to check for other signatures:

```sh
$ pdfcpu signatures validate sample2.pdf --all --full
repaired: trailer size
optimizing...

1:
       Type: form signature (authoritative, visible, signed) on page 1
     Status: signature is valid
     Reason: document has not been modified
     Signed: 2024-09-19 13:09:06 +0000
DocModified: false
    Details:
             SubFilter:      ETSI.CAdES.detached
             SignerIdentity: John Doe
             SignerName:     John Doe
             ContactInfo:
             Location:       Signature Box
             Reason:         Signature
             SigningTime:    2024-09-19 13:09:06 +0000
             Field:          Signature1
     Signer:
             Timestamp:      2024-09-19 13:10:03 +0000
             LTVEnabled:     false
             PAdES:          B-T
             Certified:      false
             Authoritative:  true
             Certificate:
                             Subject:    John Doe
                             Issuer:     a-sign-premium-mobile-05
                             SerialNr:   614a81f67
                             Valid From: 2023-01-04 10:39:36 +0000
                             Valid Thru: 2028-01-04 10:39:36 +0000
                             Expired:    false
                             Qualified:  true
                             CA:         false
                             Usage:
                             Version:    3
                             SignAlg:    ECDSA
                             Key Size:   256 bits
                             SelfSigned: false
                             Trust:      Status: ok
                                        Reason: cert chain resolved by local trust store
                             Revocation: Status: ok
                                         Reason: not revoked (OCSP check ok)
             RootCA:
                             Subject:    a-sign-premium-mobile-05
                             Issuer:     A-Trust-Root-05
                             SerialNr:   36a009c2
                             Valid From: 2022-12-19 09:15:01 +0000
                             Valid Thru: 2029-07-10 07:15:01 +0000
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
```

Using `-all` reveals that there is only one signature.   
The signature contains a single signer, which is the expected behavior for ETSI CAdES-detached signatures.

We see the certificate chain resolved against pdfcpu's local certificate store, and also that the certificate is not expired and the available OCSP responder returned a non-revoked status.

We can see timestamp evidence, which pdfcpu reports as B-T evidence. This could also be due to a separate DTS (Document Timestamp Signature).

<br>

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

In order to see the details for both signatures, you need to supply `--all` and `--full`.
(Using a combination of short flags also works: `-af`)
This form signature may contain B-T or higher evidence, such as B-LT or B-LTA evidence.
We skip this because it produces a rather long output.
The `trusted` label shown for the document timestamp is pdfcpu's current CLI output for a successfully processed DTS and should be read in the context of the OSS limitations described above.

<br>

At last, we take a look at a PDF with a usage rights signature.
This is not a signature in the traditional sense, but rather a signed definition of permissions that PDF processors should obey.
For example, you can use usage rights to explicitly allow saving a filled form.

```sh
$ pdfcpu signatures validate usageRights.pdf --all
optimizing...

1 usage rights signature (invisible, signed)
   Status: validity of the signature is unknown
   Reason: signers certificate chain is not in the configured local trusted certificate store
   Signed: 2022-12-15 17:08:57 +0000
```

Using `-all` reveals that there is only one signature.  

Let's take a detailed look at what is going on here:

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
                             SerialNr:   101357a46c30d17b2f7d64b453c0818
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
                             SerialNr:   fa8b6547b89e6d2068975cd8b9b89e2
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
                             SerialNr:   0df12f5f57a7c3e1b002d893270cdde1
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
             Problems:       certificate verification failed for serial="101357a46c30d17b2f7d64b453c0818": x509: certificate signed by unknown authority
```

A problem points to missing intermediate or root certificates. 
The certificate is therefore **not trusted**.

We can also see that the certificate is not expired and that the available CRL check did not report it as revoked.

Conclusion: If you import the missing certificates using [pdfcpu certificates import](/core/certs), pdfcpu should be able to complete its local certificate-chain check for this usage rights signature.

> **Note:**  
> This command only checks the **usage rights signature** itself.
> Any violation of **usage rights** defined as UR3 transform parameters are not checked at the moment (see [limitations](#limitations)).

<br>

Validate a signed PDF streamed from S3:

```sh
$ aws s3 cp s3://acme-signing/executed.pdf - \
   | pdfcpu signatures validate -
```

<br>

Remove signatures from a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-signing/executed.pdf - \
   | pdfcpu signatures remove - - \
   | aws s3 cp - s3://acme-signing/executed-unsigned.pdf
```
