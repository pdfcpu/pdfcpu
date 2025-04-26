---
layout: default
---

# Reset Page Mode

This command resets the configured page mode for a PDF file.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagemode reset inFile
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

Reset page mode for `test.pdf`:

```sh
$ pdfcpu pagemode reset test.pdf
$ pdfcpu pagemode list test.pdf
No page mode set, PDF viewers will default to "UseNone"
```
