---
layout: default
---

# Remove Keywords

This command removes keywords from a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords remove [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile [keyword...]
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | verbose logging   | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

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