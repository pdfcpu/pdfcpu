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