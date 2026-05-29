---
layout: default
title: "List Annotations"
---

# List Annotations

This command prints a list of annotations by type for selected pages.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu annotations list inFile [flags]
```

<br>

### Flags

| name                             | description     | required
|:---------------------------------|:----------------|---------
| json                             | output JSON     | no
| [p(ages)](/getting_started/page_selection) | selected pages | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes

<br>

## Examples

List all annotations of test.pdf:
```
$ pdfcpu annotations list test.pdf
4 annotations available

Page 3:

  Link:
     obj#                 rect id              content
    ==================================================
      539 (230, 610, 365, 625)    https://www.abc.com/
      538 (271, 637, 326, 693)    https://www.def.com/

Page 4:

  Link:
     obj#                 rect id    content
    ========================================
      563 ( 70, 354, 197, 377)    (internal)
      575 ( 70, 634, 120, 657)    (internal)
```

<br>

List annotations for a streamed PDF:

```sh
$ aws s3 cp s3://acme-redaction/review.pdf - \
   | pdfcpu annotations list -
```
