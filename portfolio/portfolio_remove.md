---
layout: default
---

# Remove Portfolio Entries

This command removes previously added entries from a PDF portfolio. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio remove inFile [file...]
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
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