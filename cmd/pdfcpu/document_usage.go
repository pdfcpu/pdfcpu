/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

const (
	usageLongValidate = `Check inFile for specification compliance.

      mode ... validation mode
     links ... check for broken links
  optimize ... optimize resources (fonts, forms, images)
    inFile ... input PDF file, use - to read from stdin

The validation modes are:
    strict ... validate against PDF 32000-1:2008 (PDF 1.7) and rudimentary against PDF 32000:2 (PDF 2.0)
   relaxed ... (default) like strict but doesn't complain about common spec violations.

Validation turns off optimization unless in verbose mode.
You can enforce optimization using --opt=true (or just --opt).

Pipeline example:
   aws s3 cp s3://acme-invoices/invoice.pdf - \
      | pdfcpu validate -`

	usageLongOptimize = `Read inFile, remove redundant page resources like embedded fonts and images and write the result to outFile.

     stats ... append a stats line to a csv file with information about the usage of root and page entries.
               useful for batch optimization and debugging PDFs.
    inFile ... input PDF file, use - to read from stdin
   outFile ... output PDF file, use - to write to stdout

Pipeline example:
   aws s3 cp s3://acme-contracts/master.pdf - \
      | pdfcpu optimize - - \
      | aws s3 cp - s3://acme-contracts/optimized/master.pdf`

	usageLongInfo = `Print info about a PDF file.

   pages ... Please refer to "pdfcpu selectedpages"
   fonts ... include font info
    json ... output JSON
  inFile ... a list of PDF input files, use - to read from stdin

Pipeline example using a file from a pdfcpu Git checkout:
   cat pkg/testdata/go.pdf \
      | pdfcpu info --json - \
      | jq '.infos[] \
      | {source, pageCount}'`

	usageLongCreate = `Create page content corresponding to declarations in inFileJSON.
Append new page content to existing page content in inFile and write result to outFile.
If inFile is absent outFile will be overwritten.

   inFileJSON ... input json file
       inFile ... optional input PDF file, use - to read from stdin
      outFile ... output PDF file, use - to write to stdout

A minimalistic sample json:
{
   "pages": {
      "1": {
         "content": {
            "text": [
               {
                  "value": "Hello pdfcpu user!",
                  "anchor": "center",
                  "font": {
                     "name": "Helvetica",
                     "size": 12
                   }
               }
            ]
         }
      }
   }
}

For more info on JSON syntax and examples in a pdfcpu Git checkout:
   pkg/testdata/json/*
   pkg/samples/create/*

Pipeline examples:
   pdfcpu create invoice.json - \
      | aws s3 cp - s3://acme-billing/invoice.pdf

   aws s3 cp s3://acme-forms/template.pdf - \
      | pdfcpu create overlay.json - - \
      | aws s3 cp - s3://acme-forms/template-filled.pdf`

	usageLongSplit = `Generate a set of PDFs for the input file in outDir according to given span value or along bookmarks or page numbers.

      mode ... split mode (defaults to span)
    inFile ... input PDF file, use - to read from stdin
    outDir ... output directory
      span ... split span in pages (default: 1) for mode "span"
    pageNr ... split before a specific page number for mode "page"

The split modes are:

      span     ... Split into PDF files with span pages each (default).
                   span itself defaults to 1 resulting in single page PDF files.

      bookmark ... Split into PDF files representing sections defined by existing bookmarks.
                   Assumption: inFile contains an outline dictionary.

      page     ... Split before specific page numbers.

Eg. pdfcpu split test.pdf .      (= pdfcpu split -m span test.pdf . 1)
      generates:
         test_1.pdf
         test_2.pdf
         etc.

    pdfcpu split test.pdf . 2    (= pdfcpu split -m span test.pdf . 2)
      generates:
         test_1-2.pdf
         test_3-4.pdf
         etc.

    pdfcpu split -m bookmark test.pdf .
      generates:
         test_bm1Title_1-4.pdf
         test_bm2Title.5-7-pdf
         etc.

    pdfcpu split -m page test.pdf . 2 4 10
      generates:
         test_1.pdf
         test_2-3.pdf
         test_4-9.pdf
         test_10-20.pdf

Pipeline example:
   aws s3 cp s3://acme-reports/board-pack.pdf - \
      | pdfcpu split -m page - ./board-pack 10 25`

	usageLongTrim = `Generate a trimmed version of inFile for selected pages.

     pages ... Please refer to "pdfcpu selectedpages"
    inFile ... input PDF file, use - to read from stdin
   outFile ... output PDF file, use - to write to stdout

Pipeline example:
   aws s3 cp s3://acme-cases/filing.pdf - \
      | pdfcpu trim -p 1-12 - - \
      | aws s3 cp - s3://acme-cases/filing-trimmed.pdf
`

	usageLongCollect = `Create custom sequence of selected pages.

        pages ... Please refer to "pdfcpu selectedpages"
       inFile ... input PDF file, use - to read from stdin
      outFile ... output PDF file, use - to write to stdout

Pipeline example:
   aws s3 cp s3://acme-dataroom/deck.pdf - \
      | pdfcpu collect -p 1,3,5-7 - - \
      | aws s3 cp - s3://acme-dataroom/executive-extract.pdf
  `

	usageLongMerge = `Concatenate a sequence of PDFs/inFiles into outFile.

          mode ... merge mode (defaults to create)
          sort ... sort inFiles by file name
     bookmarks ... create bookmarks
 bookmark-mode ... bookmark mode: wrap|preserve (defaults to wrap)
       divider ... insert blank page between merged documents
      optimize ... optimize before writing (default: true)
       outFile ... output PDF file, use - to write to stdout
        inFile ... a list of PDF files subject to concatenation.
                   use - to read from stdin for the first inFile in create mode only

The merge modes are:
    create ... outFile will be created and possibly overwritten (default).
    append ... if outFile does not exist, it will be created (like in default mode).
               if outFile already exists, inFiles will be appended to outFile.
       zip ... zip inFile1 and inFile2 into outFile (which will be created and possibly overwritten).

Skip bookmark creation: -b=false or --bookmarks=false

Preserve input bookmark trees without filename wrapper bookmarks: --bookmark-mode preserve

Skip optimization before writing: --opt=false

Pipeline examples:
   pdfcpu merge - quarterly/*.pdf \
      | aws s3 cp - s3://acme-reports/quarterly/merged.pdf

   aws s3 cp s3://acme-reports/cover.pdf - \
      | pdfcpu merge - - chapter1.pdf chapter2.pdf \
      | aws s3 cp - s3://acme-reports/book.pdf`
)
