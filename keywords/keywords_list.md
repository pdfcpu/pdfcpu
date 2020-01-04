---
layout: default
---

# List Keywords

This command outputs a list of all document keywords. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords list [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile
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

 List all document keywords of `in.pdf`:

```sh
pdfcpu keyword list in.pdf
literature
contemporary
```
