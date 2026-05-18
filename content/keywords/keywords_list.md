---
layout: default
title: "List Keywords"
---

# List Keywords

This command outputs a list of all document keywords. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords list inFile [flags]
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

 List all document keywords of `in.pdf`:

```sh
$ pdfcpu keyword list in.pdf
literature
contemporary
```

<br>

List keywords for a streamed PDF:

```sh
$ aws s3 cp s3://acme-assets/brochure.pdf - \
   | pdfcpu keywords list -
```
