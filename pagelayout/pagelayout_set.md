---
layout: default
---

# Set Page Layout

This command configures the page layout that shall be used when the document is opened.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagelayout set inFile value
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------------------------
| inFile       | PDF input file      | yes
| value        | page layout mode    | yes

<br>

### Page Layouts

| name           | description
|:---------------|:-------------------------------------------------
| SinglePage     | Display one page at a time (default)
| TwoColumnLeft  | Display the pages in two columns, with odd-numbered pages on the left
| TwoColumnRight | Display the pages in two columns, with odd-numbered pages on the right
| TwoPageLeft    | Display the pages two at a time, with odd-numbered pages on the left
| TwoPageRight   | Display the pages two at a time, with odd-numbered pages on the right

<br>

## Examples

Set page layout for `test.pdf` (case agnostic):

```sh
$ pdfcpu pagelayout set test.pdf TwoColumnLeft
$ pdfcpu pagelayout list test.pdf
TwoColumnLeft
```
