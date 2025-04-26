---
layout: default
---

# List Page Layout

This command prints the page layout that shall be used when the document is opened.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagelayout list inFile
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name    | description         | required
|:--------|:--------------------|:--------------------------
| inFile  | PDF input file                             | yes


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

List page layout of `test1.pdf`:

```sh
$ pdfcpu pagelayout list test1.pdf
No page layout set, PDF viewers will default to "SinglePage"
```

List page layout of `test2.pdf`:

```sh
$ pdfcpu pagelayout list test2.pdf
TwoPageLeft
```

