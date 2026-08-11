---
layout: default
title: "Validate"
---

# Validate

Any PDF file you would like to process with pdfcpu needs to pass validation.

Validation checks the PDF structures and objects covered by pdfcpu's implemented validation rules against:

* PDF 1.7: [PDF 32000-1:2008](https://opensource.adobe.com/dc-acrobat-sdk-docs/pdfstandards/PDF32000_2008.pdf)

* PDF 2.0: basic checks against [PDF 32000-2:2020](https://www.pdfa-inc.org/product/iso-32000-2-pdf-2-0-bundle-sponsored-access/) (ongoing task)

Validation is not a certification of complete ISO 32000 compliance. PDF 2.0 coverage is basic and continuously
improving.

<br>
Validation can also check for broken links.

<br>Have a look at some [examples](#examples).

## Usage

```
pdfcpu validate inFile... [flags]
```

<br>

### Flags

| name                             | description     | required | values          |default
|:---------------------------------|:----------------|:---------|:----------------|:------
| m(ode)                           | validation mode | no       | strict, relaxed | relaxed
| l(inks)                          | check links     | no       |                 |
| opt, optimize                    | optimize resources | no    |                 |
| progress                         | print each input before validation | no |          |

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin | yes

<br>

#### Mode

##### Strict

Strict validation rejects specification violations detected by pdfcpu's implemented validation rules.

It validates against PDF 32000-1:2008 (PDF 1.7) and performs basic PDF 32000-2:2020 (PDF 2.0) checks.

##### Relaxed

Relaxed validation is the default and is intended for processing real-world PDFs.

It applies the same validation baseline as strict mode but permits a curated set of common PDF writer deviations.
Depending on the violation, pdfcpu may accept an alternate representation, tolerate a missing or inconsistent entry, or
apply a safe, unambiguous repair to the in-memory document.

Relaxed mode broadens compatibility; it does not relax security or resource limits. Unreadable input, ambiguous
corruption, unsafe structures, and violations outside the supported compatibility exceptions still fail validation.

#### Reader recovery

Validation mode controls conformance decisions, not every aspect of PDF parsing. The reader may perform bounded
recovery of low-level structures, such as damaged cross-reference information, in either mode when reconstruction is
unambiguous.

Consequently, strict validation does not mean that the original byte stream was free of defects. It means that the
document pdfcpu reconstructed passed the strict checks currently implemented.

#### Scope

Strict and relaxed are validation policies. They do not select a PDF/A profile, provide complete PDF 2.0 validation, or
constitute a signature trust, legal-validity, or regulatory-compliance assessment.

#### Progress

Use `--progress` together with `--quiet` for quiet batch validation that still identifies each input as processing starts.
Progress and validation errors are written to standard error; normal success output remains suppressed.

Without `--quiet`, validation already reports the current input and `--progress` does not print a duplicate line.

<br>

## Examples

An example using `strict` validation:
```sh
$ pdfcpu validate test.pdf --mode strict
validating(mode=strict) test.pdf ...
validation ok
```

<br>

An example using default validation:
```sh
$ pdfcpu validate test.pdf
validating(mode=relaxed) test.pdf ...
validation ok
```

<br>

Quietly validate a directory while reporting each input as processing starts:

```sh
$ pdfcpu validate --quiet --progress invoices
validating(mode=relaxed) invoices/invoice-001.pdf ...
validating(mode=relaxed) invoices/invoice-002.pdf ...
```

<br>

Check for broken links:
```sh
$ pdfcpu val PDF32000_2008.pdf -l
validating(mode=relaxed) PDF32000_2008.pdf ...
validating URIs..
...........................
Page 8: http://www.aiim.org/pdfrefdocs status=404
Page 10: http://adobe.com/go/pdf_ref_bibliography status=404
Page 10: http://www.adobe.com/go/pdf_ref_bibliography status=404
Page 11: http://www.aiim.org/pdfnotes status=404
Page 753: http://developer.apple.com/fonts/TTRefMan/ status=404
Page 754: http://www.agfamonotype.com/printer/pan1.asp status=404
Page 755: http://www.rsasecurity.com/rsalabs/node.asp?id=2125 status=404
validation error: broken links detected
```

<br>

Validate a PDF streamed from S3:

```sh
$ aws s3 cp s3://acme-invoices/invoice.pdf - \
   | pdfcpu validate -
```
