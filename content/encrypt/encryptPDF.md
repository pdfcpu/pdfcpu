---
layout: default
title: "Encrypt"
---

# Encrypt

This command encrypts `inFile` using the standard security handler as defined in [PDF 32000-2:2020](https://www.pdfa-inc.org/product/iso-32000-2-pdf-2-0-bundle-sponsored-access/). If provided the encrypted PDF will be written to `outFile` and `inFile` remains untouched. Have a look at some [examples](#examples).

**Owner Password** <br>
Opens the document without restrictions/permissions, grants full access. <br>
Also known as permission or master password. 

**User Password** <br>
Opens the document based on configured restrictions/permissions. <br>
Also known as Open Doc password.

> IMPORTANT
>
> Both passwords are needed to compute the encryption key.
>
> While the PDF specification allows an empty owner password, it is generally not recommended because it weakens security. <br>
> Hence pdfcpu's opinionated approach makes the owner password mandatory for encryption.
>
> Setting a user password remains an option, yet it is highly recommended because it adds an extra layer of protection.<br>
> Without a user password, the PDF is encrypted but openable by anyone, and many tools can remove restrictions using just the owner password.
> Moreover you may not want users who are authorized to open a document also mess with its security restrictions.

## Usage

```
pdfcpu encrypt inFile [ outFile ] [flags]
```

<br>

### Flags

| name                                            | description     | required | values         |default
|:------------------------------------------------|:----------------|:---------|:---------------|:------
| mode                             | encryption   | no              | rc4, aes | aes
| key                              | key length   | no              | rc4:40,128 aes:40,128,256        | 256
| perm                             | permissions  | no              | none, all | none

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description               | required
|:-------------|:--------------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| outFile      | PDF output file, use `-` to write to stdout     | no

<br>

#### mode

The [symmetric encryption algorithm](https://en.wikipedia.org/wiki/Symmetric-key_algorithm) to be used for encrypting and decrypting a document. The PDF standard security handler defines two algorithms to be used: 

* [RC4](https://en.wikipedia.org/wiki/RC4)
* [AES](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard)

NOTE: RC4 is considered to be insecure!

The default mode for `pdfcpu` is AES.<br>
As of 2026 AES is still considered secure and an effective federal US government standard.

NOTE: As AES-256 is the most recent algorithm the PDF specification defines, more secure algorithms will be needed and provided in a future release.

#### key

The length of the [cryptographic key](https://en.wikipedia.org/wiki/Key_(cryptography)) used for encryption and decryption.

Possible values for RC4:

* 40
* 128

Possible values for AES:

* 40
* 128
* 256

#### perm

The set of [permissions](/encrypt/perm_list) that apply once a document has been opened.

Possible values:
* `none` clears all permission bits. This is the most restrictive way of presenting an open document to a user.

* `all` sets all permission bits allowing full access to all operations that may be applied to an open document.

> [!NOTE]
> These quick primitives will be followed up by finer grained control over the permission bits in a future release.

<br>

## Examples

Encrypt `test.pdf` using the default encryption AES with a 256-bit key and the [default permissions]().
Set the owner password to `opw`. This password also known as the *master password* or the *set permissions password* may be used to change the [permissions](). Since there is no user password set any PDF Reader may open this document:

```sh
$ pdfcpu encrypt test.pdf --opw opw
writing test.pdf ...
```
<br>

Encrypt `test.pdf` using the default encryption AES with a 256-bit key and the [default permissions]().
Set the user password to `upw`. This password must be used to open the decrypted file. It is also known as the *open doc password*, then
set the owner password to `opw`:

```sh
$ pdfcpu encrypt test.pdf --upw upw --opw opw
writing test.pdf ...
```

<br>

Encrypt `test.pdf` and write the encrypted output file to `test_enc.pdf`. Use AES with a 40-bit key and [default permissions]().
Set the mandatory owner password to `opw` which will also be needed to change the permissions of `test_enc.pdf`:

```sh
$ pdfcpu encrypt test.pdf test_enc.pdf --opw opw --mode aes --key 40
writing test_enc.pdf ...
```

<br>

Encrypt `test.pdf` and write the encrypted output file to `test_enc.pdf`. Use RC4 with a 128-bit key and set all permissions for full access.
Set the user password to `upw` which will be needed to open `test_enc.pdf`, also set the owner password to `opw`:

```sh
$ pdfcpu encrypt test.pdf test_enc.pdf --upw upw --opw opw --mode rc4 --key 128 --perm all 
writing test_enc.pdf ...
```

<br>

Encrypt a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-hr/onboarding.pdf - \
   | pdfcpu encrypt --opw "$OPW" --upw "$UPW" - - \
   | aws s3 cp - s3://acme-hr/secure/onboarding.pdf
```
