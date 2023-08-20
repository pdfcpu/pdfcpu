---
layout: default
---

# Extract Fonts

## Examples

Extract all fonts from `book.pdf` into `out`:

```sh
$ pdfcpu extract -mode font book.pdf out
extracting fonts from book.pdf into out ...

$ ls out
-rwxr-xr-x   1 horstrutter  staff    68K Mar  8 12:21 Arial_21_836.ttf*
-rwxr-xr-x   1 horstrutter  staff    46K Mar  8 12:21 TT0_1_2868.ttf*
-rwxr-xr-x   1 horstrutter  staff    45K Mar  8 12:21 TT0_3_844.ttf*
-rwxr-xr-x   1 horstrutter  staff    25K Mar  8 12:21 TT1_16_854.ttf*
-rwxr-xr-x   1 horstrutter  staff    17K Mar  8 12:21 TT1_18_911.ttf*
```

<br>

Extract all fonts of pages 5-10  from `book.pdf` into `out`: 

```sh
$ pdfcpu extract -mode font -pages 5-10 book.pdf out
extracting fonts from book.pdf into out ...

$ ls out
-rwxr-xr-x   1 horstrutter  staff    32K Mar  8 12:21 TT2_21_856.ttf*
-rwxr-xr-x   1 horstrutter  staff    12K Mar  8 12:21 TT2_21_920.ttf*
-rwxr-xr-x   1 horstrutter  staff    27K Mar  8 12:21 TT2_3_838.ttf*
-rwxr-xr-x   1 horstrutter  staff    24K Mar  8 12:21 TT3_2_834.ttf*
```
