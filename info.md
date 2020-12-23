---
layout: default
---

# Info

Print information about a PDF file and its attachments.

## Usage

```
pdfcpu info [-pages selectedPages] inFile
```

<br>

### Flags

| name                                    | description       | required | values
|:----------------------------------------|:------------------|:---------|-------
| [p(ages)](getting_started/page_selection) | page selection    | no

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](getting_started/common_flags.md)          | user password   |
| [opw](getting_started/common_flags.md)          | owner password  |

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
<br>

Use the *pages* flag to include page boundaries for selected pages in your desired display unit:
```sh
pdfcpu info -u po -pages 1,2 test.pdf
pages: 1,2
         PDF version: 1.2
          Page count: 2
Page 1:
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71  = CropBox, TrimBox, BleedBox, ArtBox
Page 2:
  MediaBox (points) (0.00, 0.00, 595.27, 841.89) w=595.27 h=841.89 ar=0.71  = CropBox, TrimBox, BleedBox, ArtBox
............................................
               Title:
              Author:
             Subject:
        PDF Producer: DOC1/EMFE v4.4M0p2286 + SCR 57461
     Content creator:
       Creation date: D:20150122062117
   Modification date:
............................................
              Tagged: No
              Hybrid: No
          Linearized: No
  Using XRef streams: No
Using object streams: No
         Watermarked: No
............................................
           Encrypted: No
         Permissions: Full access
```
