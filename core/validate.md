---
layout: default
---

# Validate

This command checks `inFile` for compliance with the specification [PDF 32000-1:2008](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/PDF32000_2008.pdf) (PDF 1.7). Any PDF file you would like to process needs to pass validation. Have a look at some [examples](#examples).

## Usage

```
pdfcpu validate [-m(ode) strict|relaxed] inFile
```

<br>

### Flags

| name                             | description     | required | values          |default
|:---------------------------------|:----------------|:---------|:----------------|:------
| m(ode)                           | validation mode | no       | strict, relaxed | relaxed

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

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
