---
layout: default
---

# Add Keywords

This command adds keywords or key phrases to a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords add [-v(erbose)|vv] [-q(uiet)] [-upw userpw] [-opw ownerpw] inFile keyword...
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
| keyword      | search keyword or keyphrase | yes

<br>

## Examples

Adding a key phrase and a keyword.
Put key phrases under single quotes:

```sh
pdfcpu keywords add in.pdf 'Tom Sawyer' classic
```
