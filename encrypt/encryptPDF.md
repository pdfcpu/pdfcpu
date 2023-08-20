---
layout: default
---

# Encrypt

This command encrypts `inFile` using the standard security handler as defined in [PDF 32000-1:2008](https://www.adobe.com/content/dam/acom/en/devnet/pdf/pdfs/PDF32000_2008.pdf). If provided the encrypted PDF will be written to `outFile` and `inFile` remains untouched. Have a look at some [examples](#examples).

## Usage

```
pdfcpu encrypt [-mode rc4|aes] [-key 40|128|256] [-perm none|all] [-upw userpw] -opw ownerpw inFile [outFile]
```

<br>

### Flags

| name                                            | description     | required | values         |default
|:------------------------------------------------|:----------------|:---------|:---------------|:------
| mode                             | encryption   | no              | rc4, aes | aes
| key                              | key length   | no              | rc4:40,128 aes:40,128,256        | 256
| perm                             | permissions  | no              | none, all | none

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description               | required
|:-------------|:--------------------------|:--------
| inFile       | PDF input file            | yes
| outFile      | encrypted PDF output file | no

<br>

#### mode

The [symmetric encryption algorithm](https://en.wikipedia.org/wiki/Symmetric-key_algorithm) to be used for encrypting and decrypting a document. The PDF standard security handler defines two algorithms to be used: 

* [RC4](https://en.wikipedia.org/wiki/RC4)
* [AES](https://en.wikipedia.org/wiki/Advanced_Encryption_Standard)

NOTE: RC4 is considered to be insecure!

The default mode for `pdfcpu` is AES.<br>
As of 2019 AES is still considered secure and an effective federal US government standard.

NOTE: As AES-256 is the most recent algorithm the PDF 1.7 specification defines, more secure algorithms will be needed and provided in a future release.

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

The set of [permissions](perm_list.md) that apply once a document has been opened.

Possible values:
* `none` clears all permission bits. This is the most restrictive way of presenting an open document to a user.

* `all` sets all permission bits allowing full access to all operations that may be applied to an open document.

NOTE: These quick primitives will be followed up by finer grained control over the permission bits in a future release.

<br>

## Examples

Encrypt `test.pdf` using the default encryption AES with a 256-bit key and the [default permissions]().
Set the owner password to `opw`. This password also known as the *master password* or the *set permissions password* may be used to change the [permissions](). Since there is no user password set any PDF Reader may open this document:

```sh
$ pdfcpu encrypt -opw opw test.pdf
writing test.pdf ...
```

<br>

Encrypt `test.pdf` using the default encryption AES with a 256-bit key and the [default permissions]().
Set the user password to `upw`. This password must be used to open the decrypted file. It is also known as the *open doc password*, then
set the owner password to `opw`:

```sh
$ pdfcpu encrypt -upw upw -opw opw test.pdf
writing test.pdf ...
```

<br>

Encrypt `test.pdf` and write the encrypted output file to `test_enc.pdf`. Use AES with a 40-bit key and [default permissions]().
Set the mandatory owner password to `opw` which will also be needed to change the permissions of `test_enc.pdf`:

```sh
$ pdfcpu encrypt -opw opw -mode aes -key 40 test.pdf test_enc.pdf
writing test_enc.pdf ...
```

<br>

Encrypt `test.pdf` and write the encrypted output file to `test_enc.pdf`. Use RC4 with a 128-bit key and set all permissions for full access.
Set the user password to `upw` which will be needed to open `test_enc.pdf`, also set the owner password to `opw`:

```sh
$ pdfcpu encrypt -upw upw -opw opw -mode rc4 -key 128 -perm all test.pdf test_enc.pdf
writing test_enc.pdf ...
```
