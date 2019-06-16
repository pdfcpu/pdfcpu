---
layout: default
---

# Add Permissions

The PDF specification defines a set of permissions that may be set for encrypted documents.
Permissions go into effect anytime an encrypted document is opened with the *user password*.
Providing the *owner password* which is also known as the *set permissions password* or *master password* will give full access to the document.

`pdfcpu` provides minimal support for setting permissions. You can set either `all` or `none` permissions. Have a look at some [examples](#examples).

## Usage

```
pdfcpu permissions add [-v(erbose)|vv] [-perm none|all] [-upw userpw] -opw ownerpw inFile
```

<br>

### Flags

| name                             | description     | required | values    | default
|:---------------------------------|:----------------|:---------|:----------|:-------
| [verbose](../getting_started/common_flags.md) | turn on logging | no
| [vv](../getting_started/common_flags.md)      | verbose logging | no
| perm                             | permissions     | no       | none, all | none
| [upw](../getting_started/common_flags.md)     | user password   | if set
| [opw](../getting_started/common_flags.md)     | owner password  | if set

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file         | yes

<br>

## Examples

You have to provide any non empty password in order to change permissions.

For a document encrypted with just the *owner password* you have to provide `opw` to change the permissions:

```
pdfcpu encrypt -opw opw enc.pdf
writing enc.pdf ...

pdfcpu perm list enc.pdf
permission bits:            0
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))

pdfcpu perm add -perm all enc.pdf
Please provide all non-empty passwords

pdfcpu perm add -opw opw -perm all enc.pdf
adding permissions to enc.pdf ...
writing enc.pdf ...

pdfcpu perm list enc.pdf
permission bits: 111100111100
Bit  3: true (print(rev2), print quality(rev>=3))
Bit  4: true (modify other than controlled by bits 6,9,11)
Bit  5: true (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: true (add or modify annotations)
Bit  9: true (fill in form fields(rev>=3)
Bit 10: true (extract(rev>=3))
Bit 11: true (modify(rev>=3))
Bit 12: true (print high-level(rev>=3))
```

<br>

For a document encrypted with just the *user password* you have to provide `upw` to change the permissions:
```
pdfcpu encrypt -upw upw enc.pdf
writing enc.pdf ...

pdfcpu perm list -upw upw enc.pdf
permission bits:            0
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))

pdfcpu perm add -perm all enc.pdf
Please provide all non-empty passwords#

pdfcpu perm add -upw upw -perm all enc.pdf
adding permissions to enc.pdf ...
writing enc.pdf ...

pdfcpu perm list enc.pdf
Please provide the correct password

pdfcpu perm list -upw upw enc.pdf
permission bits: 111100111100
Bit  3: true (print(rev2), print quality(rev>=3))
Bit  4: true (modify other than controlled by bits 6,9,11)
Bit  5: true (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: true (add or modify annotations)
Bit  9: true (fill in form fields(rev>=3)
Bit 10: true (extract(rev>=3))
Bit 11: true (modify(rev>=3))
Bit 12: true (print high-level(rev>=3))
```

<br>

For an encrypted document that has both passwords set you have to provide both `opw` and `upw` to change the permissions:
```
pdfcpu encrypt -opw opw -upw upw enc.pdf
writing enc.pdf ...

pdfcpu perm list -opw opw enc.pdf
permission bits:            0
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))

pdfcpu perm add -perm all enc.pdf
Please provide all non-empty passwords

pdfcpu perm add -opw opw -perm all enc.pdf
Please provide the correct password

pdfcpu perm add -upw upw -perm all enc.pdf
Please provide all non-empty passwords

pdfcpu perm add -upw upw -opw opw -perm all enc.pdf
adding permissions to enc.pdf ...
writing enc.pdf ...

pdfcpu perm list -upw upw enc.pdf
permission bits: 111100111100
Bit  3: true (print(rev2), print quality(rev>=3))
Bit  4: true (modify other than controlled by bits 6,9,11)
Bit  5: true (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: true (add or modify annotations)
Bit  9: true (fill in form fields(rev>=3)
Bit 10: true (extract(rev>=3))
Bit 11: true (modify(rev>=3))
Bit 12: true (print high-level(rev>=3))
```