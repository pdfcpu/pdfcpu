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

### [Common Flags](../getting_started/common_flags)

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
$ pdfcpu portfolio add portfolio.pdf *.jpg
writing portfolio.pdf ...
```
