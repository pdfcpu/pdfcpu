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

### [Common Flags](../getting_started/common_flags)

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
$ pdfcpu keywords remove test.pdf modern
```

<br>

Remove all keywords:

```sh
$ pdfcpu keywords remove test.pdf
```