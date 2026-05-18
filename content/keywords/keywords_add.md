---
layout: default
title: "Add Keywords"
---

# Add Keywords

This command adds keywords or key phrases to a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords add inFile keyword... [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| keyword      | search keyword or keyphrase | yes

<br>

## Examples

Adding a key phrase and a keyword.
Put key phrases under single quotes:

```sh
$ pdfcpu keywords add in.pdf 'Tom Sawyer' classic
```

<br>

Add keywords while reading and writing the PDF through pipes:

```sh
$ aws s3 cp s3://acme-assets/brochure.pdf - \
   | pdfcpu keywords add - - approved campaign-2026 \
   | aws s3 cp - s3://acme-assets/brochure-tagged.pdf
```
