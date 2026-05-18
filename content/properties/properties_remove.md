---
layout: default
title: "Remove Properties"
---

# Remove Properties

This command removes properties from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties remove inFile [ name... ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| name...      | one or more property names | no

<br>

## Examples

Remove a specific property from `in.pdf`:

```sh
$ pdfcpu prop remove in.pdf dept
```

<br>

Remove all properties:

```sh
$ pdfcpu prop remove test.pdf
```
<br>

Remove properties while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-assets/brochure-described.pdf - \
   | pdfcpu properties remove - - Subject \
   | aws s3 cp - s3://acme-assets/brochure-clean-meta.pdf
```
