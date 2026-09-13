---
layout: default
title: "Change Owner Password"
---

# Change Owner Password

This command changes the password which is also known as the *set permissions password* or the *master password*. Have a look at some [examples](#examples).
 
## Usage

```
pdfcpu changeopw inFile opwOld opwNew [ outFile ] [flags]
pdfcpu changeopw inFile --opwold-file oldFile --opwnew-file newFile [ outFile ] [flags]
```

<br>

### [Common Flags](/getting_started/common_flags)

<br>

### Arguments

| name         | description            | required
|:-------------|:-----------------------|:--------
| inFile       | PDF input file, use `-` to read from stdin      | yes
| opwOld       | current owner password | yes
| opwNew       | new owner password     | yes, must not be empty!
| outFile      | PDF output file, use `-` to write to stdout | no

<br>

Both current passwords are authenticated before the change. Supply the current user password with
`--upw` or `--upw-file`; omitting it tries an empty password. The old positional password (or its file alternative)
is the current owner password being changed.

## Password files

Use both `--opwold-file` and `--opwnew-file` together, omitting the positional old and new passwords.
Use `--upw-file` to supply the other password from a file. Literal password strings remain supported.
See [Password files](/getting_started/common_flags/#password-files) for newline handling, empty passwords and conflicts.

```sh
pdfcpu changeopw input.pdf --upw-file /run/secrets/upw \
  --opwold-file /run/secrets/old-password --opwnew-file /run/secrets/new-password output.pdf
```

The argument table above describes the literal-password form.
The new owner-password file must contain a non-empty password. One trailing line ending (LF or CRLF) is allowed and ignored.

## Examples

You have to set the *owner password* when you `encrypt` a file and you can change it anytime later with `changeopw`.

Change the *owner password*:
```sh
$ pdfcpu encrypt enc.pdf --opw opw
writing enc.pdf ...

$ pdfcpu changeopw enc.pdf opw opwNew
writing enc.pdf ...
```

<br>

Change the owner password for a streamed PDF and upload the result:

```sh
$ aws s3 cp s3://acme-legal/client.pdf - \
   | pdfcpu changeopw - --upw "$UPW" "$OLD_OPW" "$NEW_OPW" - \
   | aws s3 cp - s3://acme-legal/client-rotated-opw.pdf
```
