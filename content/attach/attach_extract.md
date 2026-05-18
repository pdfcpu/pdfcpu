---
layout: default
title: "Extract Attachments"
---

# Extract Attachments

This command extracts attachments from a PDF document. 
If you want to remove an extracted document you can do this using [attach remove](/attach/attach_remove). Have a look at some [examples](#examples).

## Usage

```
pdfcpu attachments extract inFile outDir [ file... ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outDir       | output directory    | yes
| file...      | one or more attachments to be extracted | no

<br>

## Examples

Extract a specific attachment from `ledger.pdf` into `out`:

```sh
$ pdfcpu attach extract ledger.pdf out invoice1.pdf
writing out/invoice.pdf
```

<br>

Extract all attachments of `ledger.pdf` into `out`:

```sh
$ pdfcpu attach extract ledger.pdf out
writing out/invoice1.pdf
writing out/invoice2.pdf
writing out/invoice3.pdf
```

<br>

Extract an attachment from a streamed PDF into a local output directory:

```sh
$ aws s3 cp s3://acme-archive/ledger.pdf - \
   | pdfcpu attachments extract - ./attachments invoice1.pdf
```
