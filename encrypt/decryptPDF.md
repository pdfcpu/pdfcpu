---
layout: default
---

# Decrypt

This command decrypts `inFile` and removes password protection. If provided the decrypted PDF will be written to `outFile` and `ìnFile` remains untouched. Have a look at some [examples](#examples).

## Usage

```
pdfcpu decrypt [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

### Flags

| name                                          | description     | required
|:----------------------------------------------|:----------------|:--------
| [upw](../getting_started/common_flags.md)     | user password   | no
| [opw](../getting_started/common_flags.md)     | owner password  | no

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

| name         | description              | required
|:-------------|:-------------------------|:--------
| inFile       | encrypted PDF input file | yes
| outFile      | PDF output file          | no

<br>

## Examples

Decrypt a file that has only the *owner password* set. This will also reset all permissions, providing full access. You don't need to provide any password:

```sh
$ pdfcpu encrypt -opw opw test.pdf
writing test.pdf ...

$ pdfcpu decrypt test.pdf 
writing test.pdf ...
```

<br>

Decrypt a file that is protected by both the *user password* and the *owner password*. This also removes the open doc password and resets all permissions providing full access. You will need to provide either of the two passwords:

```sh
$ pdfcpu encrypt -opw opw -upw upw test.pdf
writing test.pdf ...

$ pdfcpu decrypt test.pdf
Please provide the correct password

$ pdfcpu decrypt -upw upw test.pdf 
writing test.pdf ...
```

```sh
$ pdfcpu encrypt -opw opw -upw upw test.pdf
writing test.pdf ...

$ pdfcpu decrypt test.pdf
Please provide the correct password

$ pdfcpu decrypt -opw opw test.pdf 
writing test.pdf ...
```