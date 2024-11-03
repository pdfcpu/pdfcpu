---
layout: default
---

# Extract Portfolio Entries

This command extracts entries from a PDF portfolio. 
If you want to remove an extracted entry you can do this using [portfolio remove](portfolio_remove.md). Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio extract inFile outDir [file...]
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

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
$ pdfcpu portfolio extract portfolio.pdf out sketch.pdf
```

<br>

Extract all portfolio entries of `portfolio.pdf` into `out`:

```sh
$ pdfcpu portfolio extract portfolio.pdf out
```
