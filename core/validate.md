---
layout: default
---

# Validate

Any PDF file you would like to process with pdfcpu needs to pass validation.

This command validates `inFile` against:

* PDF 1.7: [PDF 32000-1:2008](https://opensource.adobe.com/dc-acrobat-sdk-docs/pdfstandards/PDF32000_2008.pdf)

* PDF 2.0: [PDF 32000-2:2020](https://www.pdfa-inc.org/product/iso-32000-2-pdf-2-0-bundle-sponsored-access/) (ongoing task!)


<br>
Validation can also check for broken links.

<br>Have a look at some [examples](#examples).

## Usage

```
pdfcpu validate [-m(ode) strict|relaxed] [-l(inks)] inFile
```

<br>

### Flags

| name                             | description     | required | values          |default
|:---------------------------------|:----------------|:---------|:----------------|:------
| m(ode)                           | validation mode | no       | strict, relaxed | relaxed
| l(inks)                          | check links     | no       |                 |

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

#### Mode

##### Strict

This mode validates against the PDF specification covering all PDF versions up to 2.0.

##### Relaxed

This is the default mode for validation.<br>
It behaves like strict but does not complain about common seen violations of the specification by PDF writers.

<br>

## Examples

An example using `strict` validation:
```sh
pdfcpu validate -mode strict test.pdf
validating(mode=strict) test.pdf ...
validation ok
```

<br>

An example using default validation:
```sh
pdfcpu validate test.pdf
validating(mode=relaxed) test.pdf ...
validation ok
```

<br>

Check for broken links:
```sh
pdfcpu val -l PDF32000_2008.pdf
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
