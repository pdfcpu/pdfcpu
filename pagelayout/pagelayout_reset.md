---
layout: default
---

# Reset Page Layout

This command resets the configured page layout for a PDF file.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagelayout reset inFile
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

Reset page layout for `test.pdf`:
```sh
$ pdfcpu pagelayout reset test.pdf
$ pdfcpu pagelayout list test.pdf
No page layout set, PDF viewers will default to "SinglePage"
```