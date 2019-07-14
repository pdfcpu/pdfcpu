---
layout: default
---

# Validate

This command checks `inFile` for compliance with the specification [PDF 32000-1:2008](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/PDF32000_2008.pdf) (PDF 1.7). Any PDF file you would like to process needs to pass validation. Have a look at some [examples](#examples).

## Usage

```
pdfcpu validate [-v(erbose)|vv] [-q(uiet)] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Flags

| name                             | description     | required | values          |default
|:---------------------------------|:----------------|:---------|:----------------|:------
| [verbose](../getting_started/common_flags.md) | turn on logging | no       |
| [vv](../getting_started/common_flags.md)      | verbose logging | no       |
| [quiet](../getting_started/common_flags.md)   | quiet mode      | no
| mode                             | validation mode | no       | strict, relaxed | relaxed
| [upw](../getting_started/common_flags.md)     | user password   | no
| [opw](../getting_started/common_flags.md)    | owner password  | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

#### Mode

##### Strict

This mode validates against the specification [PDF 32000-1:2008](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/PDF32000_2008.pdf) covering all PDF versions up to 1.7.

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
