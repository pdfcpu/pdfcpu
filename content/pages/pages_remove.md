---
layout: default
title: "Remove Pages"
---

# Remove Pages

This command removes all selected pages from a PDF file.
Have a look at some [examples](#examples).

## Usage

```
pdfcpu pages remove inFile [ outFile ] [flags]
```

<br>

### Flags

| name                                         | description    | required
|:---------------------------------------------|:---------------|---------
| [p(ages)](/getting_started/page_selection) | selected pages | yes

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile   | PDF output file, use `-` to write to stdout     | no

<br>

## Examples

Remove pages 1-3 and 5 from `notes.pdf`:

```sh
$ pdfcpu pages remove notes.pdf --pages 1-3,5
removing pages from notes.pdf ...
writing notes_new.pdf ...
```

<br>

Remove pages from streamed input and upload the result:

```sh
$ aws s3 cp s3://acme-forms/packet.pdf - \
   | pdfcpu pages remove --pages 2 - - \
   | aws s3 cp - s3://acme-forms/packet-without-page-2.pdf
```
