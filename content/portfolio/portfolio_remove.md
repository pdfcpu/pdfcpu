---
layout: default
title: "Remove Portfolio Entries"
---

# Remove Portfolio Entries

This command removes previously added entries from a PDF portfolio. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio remove inFile [ file... ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| file...      | one or more entries to be removed | no

<br>

## Examples

Remove a specific entry from `portfolio.pdf`:

```sh
$ pdfcpu portfolio remove portfolio.pdf pdfcpu.zip
writing portfolio.pdf ...
```

<br>

Remove all portfolio entries:

```sh
$ pdfcpu portfolio remove portfolio.pdf
writing portfolio.pdf ...
```

<br>

Remove a portfolio entry while reading the PDF from stdin:

```sh
$ cat package-with-contract.pdf \
   | pdfcpu portfolio remove - contract.pdf > package-without-contract.pdf
```

<br>
