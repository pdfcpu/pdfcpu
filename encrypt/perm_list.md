---
layout: default
---

# List Permissions

The PDF specification defines a set of permissions that may be set for encrypted documents.
This command prints the current permission set. Have a look at some [examples](#examples).

## Usage

```
pdfcpu permissions list [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Common Flags

| name                             | description     | required
|:---------------------------------|:----------------|:--------
| [verbose](../getting_started/common_flags.md) | turn on logging | no
| [vv](../getting_started/common_flags.md)      | verbose logging | no
| [quiet](../getting_started/common_flags.md)   | quiet mode      | no
| [upw](../getting_started/common_flags.md)     | user password   | no
| [opw](../getting_started/common_flags.md)     | owner password  | no

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file         | yes

<br>

## Permission Bits

* There are twelve bits defined, four of which are unused.
* Bits are counted starting with bit 1 from right to left.

| value | binary         | hex
|:------|:---------------|:---
|none   | 0000 0000 0000 | x000
|print  | 1000 0000 0100 | x804
|all    | 1111 0011 1100 | xF3C


## Examples

`pdfcpu` does not require any password for listing the permissions of an encrypted document unless the *user password* is set:

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
```

<br>

If both passwords are set, you need to provide either one to list permissions:

```
$ pdfcpu encrypt -opw opw -upw upw enc.pdf
writing enc.pdf ...

$ pdfcpu perm list enc.pdf
Please provide the correct password

$ pdfcpu perm list -upw upw enc.pdf
permission bits: 000000000000 (x000)
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))

$ pdfcpu perm list -opw opw enc.pdf
permission bits: 000000000000 (x000)
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))
```

<br>

If only the *user password* is set then that's also what you need to provide:

```
$ pdfcpu encrypt -upw upw enc.pdf
writing enc.pdf ...

$ pdfcpu perm list enc.pdf
Please provide the correct password

$ pdfcpu perm list -upw upw enc.pdf
permission bits: 000000000000 (x000)
Bit  3: false (print(rev2), print quality(rev>=3))
Bit  4: false (modify other than controlled by bits 6,9,11)
Bit  5: false (extract(rev2), extract other than controlled by bit 10(rev>=3))
Bit  6: false (add or modify annotations)
Bit  9: false (fill in form fields(rev>=3)
Bit 10: false (extract(rev>=3))
Bit 11: false (modify(rev>=3))
Bit 12: false (print high-level(rev>=3))
```