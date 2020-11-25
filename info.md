---
layout: default
---

# Info

Print information about a PDF file.

## Usage

```
usage: pdfcpu info [-pages selectedPages] [-u(nits)] [-upw userpw] [-opw ownerpw] inFile
```

<br>

### Flags

| name                                    | description       | required | values
|:----------------------------------------|:------------------|:---------|-------
| [pages](getting_started/page_selection) | page selection    | no
| u(units)                                | page size units   | no       |po(ints),in(ches),cm,mm
| [upw](getting_started/common_flags.md)  | user password     | no
| [opw](getting_started/common_flags.md)  | owner password    | no

<br>

## Example

```sh
pdfcpu info -u cm test.pdf
         PDF version: 1.7
          Page count: 1
           Page size: 21.00 x 29.70 cm
.........................................
               Title:
              Author:
             Subject:
        PDF Producer: pdfcpu v0.3.2
     Content creator:
       Creation date: D:20190823010752+02'00'
   Modification date: D:20190823010752+02'00'
            Keywords: key1
                      key2
          Properties: name1 = val1
                      name2 = val2
..........................................
              Tagged: No
              Hybrid: No
          Linearized: No
  Using XRef streams: Yes
Using object streams: Yes
          Watermarks: No
..........................................
           Encrypted: No
         Permissions: Full access
```