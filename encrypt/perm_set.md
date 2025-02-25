---
layout: default
---

# Set Permissions

The PDF specification defines a set of permissions that may be set for encrypted documents.
Permissions go into effect anytime an encrypted document is opened with the *user password*.
Providing the *owner password* which is also known as the *set permissions password* or *master password* will give full access to the document.

You can set either `none`, `all` or permissions for `print`. 
You can also set your permission bits in binary or hex mode.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu permissions set [-perm n(one)|p(rint)|a(ll)|max4Hex|max12Bits] [-upw userpw] -opw ownerpw inFile
```

<br>

### Flags

| name                                      | description     | required | default
|:------------------------------------------|:----------------|:---------|:-------
| perm                                    | permission bits | no       | none
| [upw](../getting_started/common_flags.md) | user password   | if set
| [opw](../getting_started/common_flags.md) | owner password  | yes

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [-o(ffline)](../getting_started/common_flags.md)| disable http traffic |                                 | 
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [opw](../getting_started/common_flags.md)       | owner password  |
| [upw](../getting_started/common_flags.md)       | user password   |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file         | yes

<br>

## Examples

You have to provide any non empty password in order to change permissions:

```
$ pdfcpu encrypt -opw opw enc.pdf
writing enc.pdf ...

$ pdfcpu perm list enc.pdf
permission bits: 000000000000 (x000)
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))

pdfcpu perm set -perm all enc.pdf
pdfcpu: please provide the owner password with -opw

$ pdfcpu perm set -opw opw -perm all enc.pdf
adding permissions to enc.pdf ...
writing enc.pdf ...

$ pdfcpu perm list enc.pdf
permission bits: 111100111100 (xF3C)
Bit  3: true (print(rev2), print quality(rev>=3))
Bit  4: true (modify other than controlled by bits 6,9,11)
Bit  5: true (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: true (add or modify annotations)
Bit  9: true (fill in form fields(rev>=3)
Bit 10: true (extract(rev>=3))
Bit 11: true (modify(rev>=3))
Bit 12: true (print high-level(rev>=3))
```
