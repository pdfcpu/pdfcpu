---
layout: default
---

# Decrypt

This command decrypts `inFile` and removes password protection. If provided the decrypted PDF will be written to `outFile` and `ìnFile` remains untouched. Have a look at some [examples](#examples).

## Usage

```
usage: pdfcpu decrypt [-v(erbose)|vv] [-upw userpw] [-opw ownerpw] inFile [outFile]
```

<br>

### Flags

| name                             | description     | required
|:---------------------------------|:----------------|:--------
| [verbose](../getting_started.md) | turn on logging | no
| [vv](../getting_started.md)      | verbose logging | no
| [upw](../getting_started.md)     | user password   | no
| [opw](../getting_started.md)     | owner password  | no

<br>

### Arguments

| name         | description              | required
|:-------------|:-------------------------|:--------
| inFile       | encrypted PDF input file | yes
| outFile      | PDF output file          | no

<br>

## Matrix

The following matrix is a result of the intrinsics of how PDF encryption works:

| encrypted using | needed for decryption | use case
|:----------------|:----------------------|:-
| opw             | -                     | reset permissions
| upw             | upw                   | remove open doc password & reset permissions
| opw and upw         | opw or upw            | remove open doc password & reset permissions

<br>

## Examples

Decrypt a file that has only the *owner password* set. This will also reset all permissions, providing full access. You don't need to provide any password:

```sh
pdfcpu encrypt -opw opw test.pdf
writing test.pdf ...

pdfcpu decrypt test.pdf 
writing test.pdf ...
```

<br>

Decrypt a file that has only the *user password* set. This will remove the open doc password and also reset all permissions, providing full access. You need to provide the *user password*:

```sh
pdfcpu encrypt -upw upw test.pdf
writing test.pdf ...

pdfcpu decrypt test.pdf
Please provide the correct password

pdfcpu decrypt -upw upw test.pdf 
writing test.pdf ...
```

<br>

Decrypt a file that is protected by both the *user password* and the *owner password*. This also removes the open doc password and resets all permissions providing full access. You will need to provide either of the two passwords:

```sh
pdfcpu encrypt -opw opw -upw upw test.pdf
writing test.pdf ...

pdfcpu decrypt test.pdf
Please provide the correct password

pdfcpu decrypt -upw upw test.pdf 
writing test.pdf ...
```

```sh
pdfcpu encrypt -opw opw -upw upw test.pdf
writing test.pdf ...

pdfcpu decrypt test.pdf
Please provide the correct password

pdfcpu decrypt -opw opw test.pdf 
writing test.pdf ...
```