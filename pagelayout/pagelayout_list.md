---
layout: default
---

# List Page Layout

This command displays the configured page layout for a PDF file.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu pagelayout list inFile
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

| name    | description         | required
|:--------|:--------------------|:--------------------------
| inFile  | PDF input file                             | yes


<br>

## Examples

Display the page layout for `test1.pdf`:

```sh
$ pdfcpu pagelayout list test1.pdf
No page layout set, PDF viewers will default to "SinglePage"
```

<br>

Display the page layout for `test2.pdf`:
```sh
$ pdfcpu pagelayout list test2.pdf
TwoColumnLeft
```
