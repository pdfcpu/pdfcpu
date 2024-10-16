---
layout: default
---

# Extract Images

This command lets you extract individual images contained by pages:

Have a look at some [examples](#examples).

## Usage

```
pdfcpu images extract [-p(ages) selectedPages] -- inFile outDir
````

<br>

### Flags

| name                                         | description     | required   |
|:---------------------------------------------|:----------------|:-----------|
| [p(ages)](../getting_started/page_selection) | page selection  | no         |

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)    | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)       | user password   |
| [opw](../getting_started/common_flags.md)       | owner password  |

<br>

### Arguments

| name   | description      | required
|:-------|:-----------------|:--------
| inFile | PDF input file   | yes
| outDir | output directory | yes

## Examples

Extract all images of `book.pdf` into the current directory:

```sh
$ pdfcpu images extract book.pdf .
extracting images from book.pdf into . ...

$ ls
-rwxr-xr-x   1 horstrutter  staff    28K Mar  8 11:57 Im0_16_165.jpg*
-rw-r--r--   1 horstrutter  staff   600B Mar  8 11:57 Im1_3_36.png
-rw-r--r--   1 horstrutter  staff    93B Mar  8 12:06 Im91_22_601.png
-rw-r--r--   1 horstrutter  staff    89B Mar  8 12:06 Im181_22_716.png
-rw-r--r--   1 horstrutter  staff    93B Mar  8 12:06 Im37_22_782.jpg
-rw-r--r--   1 horstrutter  staff    89B Mar  8 12:06 Im29_22_761.png
-rw-r--r--   1 horstrutter  staff    76B Mar  8 12:06 Im124_22_539.png
-rw-r--r--   1 horstrutter  staff    16K Mar  8 12:06 Im2_22_429.jpg
-rw-r-----@  1 horstrutter  staff   537K Jun  9  2017 book.pdf
```

<br>

Extract all images of the first 5 pages of `folder.pdf` into `out`:

```sh
$ pdfcpu images extract -pages -5 folder.pdf out
pageSelection: -5
extracting images from folder.pdf into out ...

 $ ls out
-rwxr-xr-x   1 horstrutter  staff    26K Mar  8 12:10 Im0_1_2880.jpg*
-rwxr-xr-x   1 horstrutter  staff    10K Mar  8 12:10 Im0_2_7.jpg*
-rwxr-xr-x   1 horstrutter  staff   9.9K Mar  8 12:10 Im0_3_20.jpg*
-rwxr-xr-x   1 horstrutter  staff   5.1K Mar  8 12:10 Im0_4_33.jpg*
-rwxr-xr-x   1 horstrutter  staff   7.7K Mar  8 12:10 Im0_5_48.jpg*
-rwxr-xr-x   1 horstrutter  staff    11K Mar  8 12:10 Im1_2_8.jpg*
-rwxr-xr-x   1 horstrutter  staff   4.9K Mar  8 12:10 Im1_3_21.jpg*
```