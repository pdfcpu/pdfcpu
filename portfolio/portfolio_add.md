---
layout: default
---

# Add Portfolio Entries

This command adds one or more files to a PDF input file aka the PDF portfolio. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio add inFile file[,desc]...
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](getting_started/common_flags.md)          | user password   |
| [opw](getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| file...      | one or more files to be attached | yes
| desc         | description         | no

<br>

## Examples

Add pictures to a PDF portfolio for easy content delivery:

```sh
pdfcpu portfolio add portfolio.pdf *.jpg
writing portfolio.pdf ...
```
