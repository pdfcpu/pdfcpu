---
layout: default
title: "Remove Keywords"
---

# Remove Keywords

This command removes keywords from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords remove inFile [ keyword... ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| keyword...   | one or more search keywords or keyphrases | no

<br>

## Examples

Remove a specific keyword from `test.pdf`:

```sh
$ pdfcpu keywords remove test.pdf modern
```

<br>

Remove all keywords:

```sh
$ pdfcpu keywords remove test.pdf
```
<br>

Remove keywords while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-assets/brochure-tagged.pdf - \
   | pdfcpu keywords remove - - campaign-2026 \
   | aws s3 cp - s3://acme-assets/brochure-untagged.pdf
```
