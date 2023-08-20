---
layout: default
---

# Extract Content

## Examples

Extract page content in PDF syntax from `book.pdf` into `out`:

```sh
$ pdfcpu extract -mode content book.pdf out
extracting content from book.pdf into out ...

$ cd out && ls
-rwxr-xr-x   1 horstrutter  staff   9.1K Mar  8 12:27 23_791.txt*
-rwxr-xr-x   1 horstrutter  staff   4.2K Mar  8 12:27 25_824.txt*
-rwxr-xr-x   1 horstrutter  staff   1.9K Mar  8 12:27 8_147.txt*
-rwxr-xr-x   1 horstrutter  staff   9.3K Mar  8 12:27 10_173.txt*
-rwxr-xr-x   1 horstrutter  staff    12K Mar  8 12:27 18_330.txt*
-rwxr-xr-x   1 horstrutter  staff   7.2K Mar  8 12:27 19_353.txt*

$ cat 8_147.txt
BT
/P <</MCID 0 >>BDC
/CS0 cs 0 0 0  scn
/GS0 gs
/TT0 1 Tf
12 0 0 12 306 708.96 Tm
( )Tj
EMC
/P <</MCID 1 >>BDC
0 -1.15 TD
( )Tj
EMC
ET
/InlineShape <</MCID 2 >>BDC
q
107.94 692.52 396.12 -153.84 re
W* n
q
/GS1 gs
396.1199951 0 0 153.7200012 107.9400024 538.740097 cm
/Im0 Do
Q
etc..

```