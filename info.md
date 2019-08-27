---
layout: default
---

# Info

Print information about a PDF file.

## Usage

```
usage: pdfcpu info [-u(nits)] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Flags

| name                             | description       | required | values
|:---------------------------------|:------------------|:---------|-------
| u(units)                         | page size units   | no       |po(ints),in(ches),cm,mm
| [upw](../getting_started/common_flags.md)     | user password     | no
| [opw](../getting_started/common_flags.md)     | owner password    | no

<br>

## Example

```
pdfcpu info -u cm test.pdf
         PDF version: 1.7
          Page count: 1
           Page size: 21.00 x 29.70 cm
.........................................
               Title:
              Author:
             Subject:
        PDF Producer: pdfcpu v0.2.4
     Content creator:
       Creation date: D:20190823010752+02'00'
   Modification date: D:20190823010752+02'00'
..........................................
              Tagged: No
              Hybrid: No
          Linearized: No
  Using XRef streams: Yes
Using object streams: Yes
..........................................
           Encrypted: No
         Permissions: Full access
```