---
layout: default
title: "Extract Portfolio Entries"
---

# Extract Portfolio Entries

This command extracts entries from a PDF portfolio. 
If you want to remove an extracted entry you can do this using [portfolio remove](/portfolio/portfolio_remove). Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio extract inFile outDir [ file... ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
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

<br>

Extract a portfolio entry from a streamed PDF into a local output directory:

```sh
$ aws s3 cp s3://acme-dataroom/package.pdf - \
   | pdfcpu portfolio extract - ./portfolio-entries contract.pdf
```
