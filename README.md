# pdfcpu: a golang pdf processor

[![Build Status](https://travis-ci.org/hhrutter/pdfcpu.svg?branch=master)](https://travis-ci.org/hhrutter/pdfcpu)
[![GoDoc](https://godoc.org/github.com/hhrutter/pdfcpu?status.svg)](https://godoc.org/github.com/hhrutter/pdfcpu)
[![Coverage Status](https://coveralls.io/repos/github/hhrutter/pdfcpu/badge.svg?branch=master)](https://coveralls.io/github/hhrutter/pdfcpu?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/hhrutter/pdfcpu)](https://goreportcard.com/report/github.com/hhrutter/pdfcpu)
[![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://opensource.org/licenses/Apache-2.0)

![logo](resources/pdfchip3.png)

Package pdfcpu is a simple PDF processing library written in [Go](http://golang.org) supporting encryption.
It provides both an API and a CLI. Supported are all versions up to PDF 1.7 (ISO-32000).

## Status

Version: 0.1.21

Fixes: #51, #58

This release features two new commands:

* N-up
* Grid

The *N-up* command rearranges the pages of a PDF file in order to reduce its page count.<br>
This is achieved by rendering the input pages onto a grid which dimensions are defined by the supplied [N-up](https://en.wikipedia.org/wiki/N-up) value (2, 3, 4, 6, 8, 9, 12, 16).<br>
Supported are various n-Up orientations: rd(right,down)=default, dr(down,right), ld(left,down), dl(down,left)<br>
Proper rotation based on involved aspect ratios will be applied during the process. 

`pdfcpu nup out.pdf 4 in.pdf` produces a PDF-file where each page fits 4 original pages into a 2x2 grid:<br>

<p align="center">
  <img border="2" src="resources/nup4pdf.png" height="200">
</p>

The output file will use the page size of the input file unless explicitly declared by a description string like so:<br>
`pdfcpu nup 'f:A4' out.pdf 9 in.pdf`<br>

<p align="center">
  <img border="2" src="resources/nup9pdf.png" width="145">
</p>

Please refer to `pdfcpu help paper` for a list of supported paper formats.
Most well known paper size standards are supported.

`nup` also accepts a list of image files with the result of rendering all images
in N-up fashion into a PDF file using the specified paper size (default=A4):<br>

`pdfcpu nup 'f:A4L' out.pdf 4 *.jpg *.png *.tif`<br>
generates a PDF file using *A4 Landscape* where each page fits 4 images onto a 2x2 grid.
Grid border lines are rendered by default:
<p align="center">
  <img border="2" src="resources/nup4img.png" height="200">
</p>

A single image input file will produce a single page PDF with the image N-up'ed accordingly, eg.<br>
`pdfcpu nup 'f:Ledger, b:off, m:0' out.pdf 16 logo.jpg`<br>
Both grid borders and margins are suppressed in this example and the output format is *Ledger*:
<p align="center">
  <img border="2" src="resources/nup16img.png" height="200">
</p>
<br>

The *grid* command rearranges the pages of a PDF file for enhanced reading experience.
The page size of the output file is a grid of specified dimensions in original page units.
Pages may be big but that's ok since they are not supposed to be printed. One use case mentioned by the
community was to produce PDF files for source code listings eg. in the form of 10x1 grid pages:

`pdfcpu grid 'b:off' out.pdf 1 4 in.pdf`<br>
rearranges pages of in.pdf into 1x4 grids and writes the result to out.pdf using the default orientation.<br>
The output page size is the result of a 1(hor)x4(vert) page grid using in.pdf's page size:
<p align="center">
  <img border="1" src="resources/gridpdf.png" height="200">
</p>

When applied to image files this command produces photo galleries of arbitrary dimensions in PDF form.<br>
`pdfcpu grid 'd:500 500, m:20, b:off' out.pdf 5 2 *.jpg`<br>
arranges imagefiles onto a 5x2 page grid and writes the result to out.pdf using a grid cell size of 500x500:
<p align="center">
  <img border="1" src="resources/gridimg.png" height="200">
</p>


## Motivation

This is an effort to build a PDF processing library from the ground up written in Go with strong support for batch processing via a rich command line. Over time `pdfcpu` aims to support the standard range of PDF processing features and also any interesting use cases that may present themselves along the way.

One example is reducing the size of large PDF files for mass mailings by optimization to the bare minimum. This can be achieved by analyzing a PDF's cross reference table, removing redundant embedded resources like font files or images and by always writing back the file maxing out PDF compression. I also wanted to have my own swiss army knife for PDFs written entirely in [Go](http://golang.org) that allows me to trim, split, stamp and merge PDF content.

## Features

* Validate (validates PDF files up to version 7.0)
* Read (builds xref table from PDF file)
* Write (writes xref table to PDF file)
* Optimize (gets rid of redundancies like duplicate fonts, images)
* Split (split multi-page PDF into several PDFs according to split span)
* Merge (a set of PDF files into one consolidated PDF file)
* Extract Images (extract all embedded images of a PDF file into a given dir)
* Extract Fonts (extract all embedded fonts of a PDF file into a given dir)
* Extract Pages (extract specific pages into a given dir)
* Extract Content (extract the PDF-Source into given dir)
* Extract Metadata (extract XML metadata)
* Trim (generate a custom version of a PDF file including selected pages)
* Stamp/Watermark selected pages with text, image or PDF page
* Import convert/import images into PDF
* N-up (rearrange pages/images into grid page layout for reduced number of pages)
* Grid (rearrange pages/images into grid page layout for enhanced browsing experience)
* Rotate selected pages
* Manage (add,remove,list,extract) embedded file attachments
* Encrypt (sets password protection)
* Decrypt (removes password protection)
* Change user/owner password
* Manage (add,list) user access permissions

## Demo Screencast (this is an older version with a smaller command set)

[![asciicast](resources/demo.png)](https://asciinema.org/a/P5jaAo9kgZXKj2iSA1OqIdLAU)

## Installation

Required build version: go1.9 and up

`go get github.com/hhrutter/pdfcpu/cmd/...`

## Usage

    pdfcpu validate [-verbose] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile
    pdfcpu optimize [-verbose] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]
    pdfcpu split [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [span]
    pdfcpu merge [-verbose] outFile inFile...
    pdfcpu extract [-verbose] -mode image|font|content|page|meta [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir
    pdfcpu trim [-verbose] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile outFile
    
    pdfcpu stamp [-verbose] -pages pageSelection description inFile [outFile]
    pdfcpu watermark [-verbose] -pages pageSelection description inFile [outFile]
    pdfcpu import [-v(erbose)|vv] [description] outFile imageFile...
    pdfcpu nup [-v(erbose)|vv] [-pages pageSelection] [description] outFile n inFile|imageFiles...
    pdfcpu grid [-v(erbose)|vv] [-pages pageSelection] [description] outFile m n inFile|imageFiles...
    pdfcpu rotate [-v(erbose)|vv] [-pages pageSelection] inFile rotation

    pdfcpu attach list [-verbose] [-upw userpw] [-opw ownerpw] inFile
    pdfcpu attach add [-verbose] [-upw userpw] [-opw ownerpw] inFile file...
    pdfcpu attach remove [-verbose] [-upw userpw] [-opw ownerpw] inFile [file...]
    pdfcpu attach extract [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [file...]

    pdfcpu encrypt [-verbose] [-mode rc4|aes] [-key 40|128] [-perm none|all] [-upw userpw] [-opw ownerpw] inFile [outFile]
    pdfcpu decrypt [-verbose] [-upw userpw] [-opw ownerpw] inFile [outFile]
    pdfcpu changeupw [-verbose] [-opw ownerpw] inFile upwOld upwNew
    pdfcpu changeopw [-verbose] [-upw userpw] inFile opwOld opwNew

    pdfcpu perm list [-verbose] [-upw userpw] [-opw ownerpw] inFile
    pdfcpu perm add [-verbose] [-perm none|all] [-upw userpw] -opw ownerpw inFile

    pdfcpu version

 [Please read the documentation](https://godoc.org/github.com/hhrutter/pdfcpu)

## Contributing

### What

* Please open an issue if you find a bug or want to propose a change.
* Feature requests - always welcome!
* Bug fixes - always welcome!
* PRs - also welcome, although I can't promise a merge-in right now.
* `pdfcpu` is stable but still _alpha_ and occasionally undergoing heavy changes.

### How

* If you want to report a bug please attach the *very verbose* (`pdfcpu cmd -vv ...`) output and ideally a test PDF that you can share.
* Always make sure your contribution is based on the latest commit.
* Please sign your commits.
* Please sign the [CLA](https://cla-assistant.io/hhrutter/pdfcpu) before you submit a PR.

## Contributors

Thanks goes to these wonderful people:
<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
| [<img src="https://avatars1.githubusercontent.com/u/11322155?v=4" width="100px;"/><br /><sub><b>Horst Rutter</b></sub>](https://github.com/hhrutter)<br /> |[<img src="https://avatars0.githubusercontent.com/u/5140211?v=4" width="100px;"/><br /><sub><b>haldyr</b></sub>](https://github.com/haldyr)<br /> | [<img src="https://avatars3.githubusercontent.com/u/20608155?v=4" width="100px;"/><br /><sub><b>Vyacheslav</b></sub>](https://github.com/SimePel)<br /> | [<img src="https://avatars1.githubusercontent.com/u/617459?v=4" width="100px;"/><br /><sub><b>Erik Unger</b></sub>](https://github.com/ungerik)<br /> ||||
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
<!-- ALL-CONTRIBUTORS-LIST:END - Do not remove or modify this section -->

## Code of Conduct

Please note that this project is released with a Contributor [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Disclaimer

Usage of `pdfcpu` assumes you know about and respect all copyrights of any PDF content you may be processing. This applies to the PDF files as such, their content and in particular all embedded resources like font files or images. Credit goes to [Renee French](https://instagram.com/reneefrench) for creating our beloved Gopher.

## License

Apache-2.0

## Powered By

<p align="center">
  <a href="https://golang.org"> <img src="resources/Go-Logo_Aqua.png" width="200"> </a>
</p>
