---
layout: default
---

# List Portfolio Entries

A PDF portfolio entry is any file previously added to a PDF portfolio. This command outputs a list of all entries. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio list [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | verbose logging   | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all portfolio entries embedded into `portfolio.pdf`. You may add any kind of file to a PDF portfolio:

```sh
pdfcpu portfolio list portfolio.pdf
forest.jpg
pdfcpu.zip
invoice.pdf
```
