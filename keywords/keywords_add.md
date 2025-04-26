---
layout: default
---

# Add Keywords

This command adds keywords or key phrases to a PDF document. Have a look at some [examples](#examples).

## Usage

```
pdfcpu keywords add inFile keyword...
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| keyword      | search keyword or keyphrase | yes

<br>

## Examples

Adding a key phrase and a keyword.
Put key phrases under single quotes:

```sh
$ pdfcpu keywords add in.pdf 'Tom Sawyer' classic
```
