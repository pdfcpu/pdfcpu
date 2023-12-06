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

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------------------------
| inFile       | PDF input file      | yes
| value        | page layout mode    | yes

<br>

### Page Layout Modes

| name           | description
|:---------------|:-------------------------------------------------
| SinglePage     | Display one page at a time (default)
| TwoColumnLeft  | Display the pages in two columns, with odd-numbered pages on the left
| TwoColumnRight | Display the pages in two columns, with odd-numbered pages on the right
| TwoPageLeft    | Display the pages two at a time, with odd-numbered pages on the left
| TwoPageRight   | Display the pages two at a time, with odd-numbered pages on the right

<br>

## Examples

Set pagelayout for `test.pdf`:

```sh
$ pdfcpu pagelayout set test.pdf TwoColumnLeft
$ pdfcpu pagelayout list test.pdf
TwoColumnLeft
```
