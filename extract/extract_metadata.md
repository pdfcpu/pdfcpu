---
layout: default
---

# Extract Metadata

## Examples

Extract XML-Metadata from `book.pdf` into the current directory:

```sh
pdfcpu extract -mode meta book.pdf .
extracting metadata from book.pdf into . ...

ls
-rwxr-xr-x   1 horstrutter  staff    45K Mar  8 12:40 177_Catalog.txt*
-rw-r-----@  1 horstrutter  staff   537K Jun  9  2017 book.pdf

cat 177_Catalog.txt
<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="3.1.1-111">
   <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
      <rdf:Description rdf:about=""
            xmlns:dc="http://purl.org/dc/elements/1.1/">
         <dc:format>image/epsf</dc:format>
         <dc:title>
            <rdf:Alt>
               <rdf:li xml:lang="x-default">Print</rdf:li>
            </rdf:Alt>
         </dc:title>
      </rdf:Description>
      <rdf:Description rdf:about=""
            xmlns:xap="http://ns.adobe.com/xap/1.0/">
         <xap:MetadataDate>2011-12-23T14:44:17+01:00</xap:MetadataDate>
         <xap:ModifyDate>2011-12-23T14:44:17+01:00</xap:ModifyDate>
         <xap:CreateDate>2011-12-23T14:44:17+01:00</xap:CreateDate>

etc...
```