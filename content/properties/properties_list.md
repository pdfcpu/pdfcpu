---
layout: default
title: "List Properties"
---

# List Properties

This command outputs a list of all properties. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties list inFile [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes

<br>

## Examples

 List all document properties of `in.pdf`:

```sh
$ pdfcpu properties list in.pdf
dept = hr
group = 3
```

<br>

List properties for a streamed PDF:

```sh
$ aws s3 cp s3://acme-assets/brochure.pdf - \
   | pdfcpu properties list -
```
