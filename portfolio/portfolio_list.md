---
layout: default
---

# List Portfolio Entries

A PDF portfolio entry is any file previously added to a PDF portfolio. This command outputs a list of all entries. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio list inFile
```

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

## Examples

 List all portfolio entries embedded into `portfolio.pdf`. You may add any kind of file to a PDF portfolio:

```sh
$ pdfcpu portfolio list portfolio.pdf
forest.jpg
pdfcpu.zip
invoice.pdf
```
