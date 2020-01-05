---
layout: default
---

# List Properties

This command outputs a list of all properties. Have a look at some [examples](#examples).

## Usage

```
pdfcpu properties list [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
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

<br>

## Examples

 List all document properties of `in.pdf`:

```sh
pdfcpu properties list in.pdf
dept = hr
group = 3
```
