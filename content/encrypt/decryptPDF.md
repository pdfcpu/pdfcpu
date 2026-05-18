---
layout: default
title: "Decrypt"
---

# Decrypt

This command decrypts `inFile` and removes password protection. If provided the decrypted PDF will be written to `outFile` and `ìnFile` remains untouched. Have a look at some [examples](#examples).

## Usage

```
pdfcpu decrypt inFile [ outFile ] [flags]
```

<br>

### Flags

| name                                          | description     | required
|:----------------------------------------------|:----------------|:--------
| [upw](/getting_started/common_flags)     | user password   | no
| [opw](/getting_started/common_flags)     | owner password  | no

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description              | required
|:-------------|:-------------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout     | no

<br>

## Examples

Decrypt a file that has only the *owner password* set. This will also reset all permissions, providing full access. You don't need to provide any password:

```sh
$ pdfcpu encrypt test.pdf --opw opw
writing test.pdf ...

$ pdfcpu decrypt test.pdf 
writing test.pdf ...
```

<br>

Decrypt a file that is protected by both the *user password* and the *owner password*. This also removes the open doc password and resets all permissions providing full access. You will need to provide either of the two passwords:

```sh
$ pdfcpu encrypt test.pdf --opw opw --upw upw
writing test.pdf ...

$ pdfcpu decrypt test.pdf
Please provide the correct password

$ pdfcpu decrypt test.pdf --upw upw 
writing test.pdf ...
```

<br>

Decrypt a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-hr/secure/onboarding.pdf - \
   | pdfcpu decrypt --upw "$UPW" - - \
   | aws s3 cp - s3://acme-hr/plain/onboarding.pdf
```
