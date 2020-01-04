---
layout: default
---

# Remove Portfolio Entries

This command removes previously added entries from a PDF portfolio. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio remove [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [file...]
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
| file...      | one or more entries to be removed | no

<br>

## Examples

Remove a specific entry from `portfolio.pdf`:

```sh
pdfcpu portfolio remove portfolio.pdf pdfcpu.zip
writing portfolio.pdf ...
```

<br>

Remove all portfolio entries:

```sh
pdfcpu portfolio remove portfolio.pdf
writing portfolio.pdf ...
```