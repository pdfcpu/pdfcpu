---
layout: default
---

# Add Portfolio Entries

This command embeds one or more files by attaching them to a PDF input file. Have a look at some [examples](#examples).

## Usage

```
pdfcpu portfolio add [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile file...
```

<br>

### Flags

| name                                          | description       | required
|:----------------------------------------------|:------------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging   | no
| [vv](../getting_started/common_flags.md)      | verbose logging   | no
| [quiet](../getting_started/common_flags.md)   | quiet mode        | no
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| file...      | one or more files to be attached | yes

<br>

## Examples

Add pictures to a PDF portfolio for easy content delivery:

```sh
pdfcpu portfolio add portfolio.pdf *.jpg
writing portfolio.pdf ...
```
