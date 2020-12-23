---
layout: default
---

# Remove Keywords

This command removes keywords from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords remove inFile [keyword...]
```

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](getting_started/common_flags.md)          | user password   |
| [opw](getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| keyword...   | one or more search keywords or keyphrases | no

<br>

## Examples

Remove a specific keyword from `test.pdf`:

```sh
pdfcpu keyword remove test.pdf modern
```

<br>

Remove all keywords:

```sh
pdfcpu key remove test.pdf
```