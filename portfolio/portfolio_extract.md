---
layout: default
---

# Extract Portfolio Entries

This command extracts entries from a PDF portfolio. 
If you want to remove an extracted entry you can do this using [portfolio remove](portfolio_remove.md). Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio extract [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile outDir [file...]
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | quiet mode        | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| outDir       | output directory    | yes
| file...      | one or more entries to be extracted | no

<br>

## Examples

Extract a specific portfolio entry from `portfolio.pdf` into `out`:

```sh
pdfcpu portfolio extract portfolio.pdf out sketch.pdf
```

<br>

Extract all portfolio entries of `portfolio.pdf` into `out`:

```sh
pdfcpu portfolio extract portfolio.pdf out
```
