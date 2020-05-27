# pdfcpu: a Go PDF processor

[![Build Status](https://travis-ci.org/pdfcpu/pdfcpu.svg?branch=master)](https://travis-ci.org/pdfcpu/pdfcpu)
[![GoDoc](https://godoc.org/github.com/pdfcpu/pdfcpu?status.svg)](https://godoc.org/github.com/pdfcpu/pdfcpu)
[![Coverage Status](https://coveralls.io/repos/github/pdfcpu/pdfcpu/badge.svg?branch=master)](https://coveralls.io/github/pdfcpu/pdfcpu?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/pdfcpu/pdfcpu)](https://goreportcard.com/report/github.com/pdfcpu/pdfcpu)
[![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://opensource.org/licenses/Apache-2.0)
[![Latest release](https://img.shields.io/github/release/pdfcpu/pdfcpu.svg)](https://github.com/pdfcpu/pdfcpu/releases)

<img src="resources/logoSmall.png" width="150">

pdfcpu is a PDF processing library written in [Go](http://golang.org) supporting encryption.
It provides both an API and a CLI. Supported are all versions up to PDF 1.7 (ISO-32000).

## Motivation

This is an effort to build a comprehensive PDF processing library from the ground up written in Go. Over time pdfcpu aims to support the standard range of PDF processing features and also any interesting use cases that may present themselves along the way.

<p align="center">
  <kbd><a href="https://pdfcpu.io/core/grid"><img src="resources/gridpdf.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/watermark"><img src="resources/wmi1abs.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/nup"><img src="resources/nup9pdf.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/4exp.png" height="150"></a></kbd><br><br>
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/sti.png" height="150"></a></kbd>&nbsp;
  <kbd><img src="resources/hold3.png" height="150"></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/watermark"><img src="resources/wmi4.png" height="150"></a></kbd>&nbsp;<br><br>
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/stp.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/grid"><img src="resources/gridimg.png" height="150"></a></kbd>
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/stRoundBorder.png" height="150"></a></kbd>
</p>

## Focus

The main focus lies on strong support for batch processing and scripting via a rich command line. At the same time pdfcpu wants to make it easy to integrate PDF processing into your Go based backend system by providing a robust command set.

## Command Set

* [attachments](https://pdfcpu.io/attach/attach)
* [change owner password](https://pdfcpu.io/encrypt/change_opw)
* [change user password](https://pdfcpu.io/encrypt/change_upw)
* [collect](https://pdfcpu.io/core/collect)
* [decrypt](https://pdfcpu.io/encrypt/decryptPDF)
* [encrypt](https://pdfcpu.io/encrypt/encryptPDF)
* [extract](https://pdfcpu.io/extract/extract)
* [fonts](https://pdfcpu.io/fonts/fonts)
* [grid](https://pdfcpu.io/core/grid)
* [import](https://pdfcpu.io/generate/import)
* [info](https://pdfcpu.io/info)
* [keywords](https://pdfcpu.io/keywords/keywords)
* [merge](https://pdfcpu.io/core/merge)
* [nup](https://pdfcpu.io/core/nup)
* [optimize](https://pdfcpu.io/core/optimize)
* [pages](https://pdfcpu.io/pages/pages)
* [permissions](https://pdfcpu.io/encrypt/perm_add)
* [portfolio](https://pdfcpu.io/portfolio/portfolio)
* [properties](https://pdfcpu.io/properties/properties)
* [rotate](https://pdfcpu.io/core/rotate)
* [split](https://pdfcpu.io/core/split)
* [stamp](https://pdfcpu.io/core/stamp)
* [trim](https://pdfcpu.io/core/trim)
* [validate](https://pdfcpu.io/core/validate)
* [watermark](https://pdfcpu.io/core/watermark)

## Documentation

* The main entry point is [pdfcpu.io](https://pdfcpu.io).
* For CLI examples also go to [pdfcpu.io](https://pdfcpu.io). There you will find explanations of all the commands and their parameters.
* For API examples of all pdfcpu operations please refer to [GoDoc](https://godoc.org/github.com/pdfcpu/pdfcpu/pkg/api).

### GoDoc

* [pdfcpu package](https://godoc.org/github.com/pdfcpu/pdfcpu)
* [pdfcpu API](https://godoc.org/github.com/pdfcpu/pdfcpu/pkg/api)
* [pdfcpu CLI](https://godoc.org/github.com/pdfcpu/pdfcpu/pkg/cli)

## Reminder

* Always make sure your work is based on the latest commit!<br>
* pdfcpu is still *Alpha* - bugfixes are committed on the fly and will be mentioned in the next release notes.<br>
* Follow [pdfcpu](https://twitter.com/pdfcpu) for news and release announcements.
* For quick questions or discussions get in touch on the [Gopher Slack](https://invite.slack.golangbridge.org/) in the #pdfcpu channel.


## Demo Screencast

(using older version with a smaller command set)

[![asciicast](resources/demo.png)](https://asciinema.org/a/P5jaAo9kgZXKj2iSA1OqIdLAU)

## Installation

### Download
Get the latest binary [here](https://github.com/pdfcpu/pdfcpu/releases).


### Using GOPATH

Required go version for building: go1.14 and up

```
go get github.com/pdfcpu/pdfcpu/cmd/...
cd $GOPATH/src/github.com/pdfcpu/pdfcpu/cmd/pdfcpu
go install
pdfcpu version
```

### Using Go Modules

```
git clone https://github.com/pdfcpu/pdfcpu
cd pdfcpu/cmd/pdfcpu
go install
pdfcpu version
```

### Using Homebrew (macOS)
```
brew install pdfcpu
pdfcpu version
```

### Run in a Docker container

```
docker build -t pdfcpu .
# mount current folder into container to process local files
docker run -it --mount type=bind,source="$(pwd)",target=/app pdfcpu ./pdfcpu validate -mode strict /app/pdfs/a.pdf
```

## Contributing

### What

* Please open an issue if you find a bug or want to propose a change.
* Feature requests - always welcome!
* Bug fixes - always welcome!
* PRs - anytime!
* pdfcpu is stable but still *Alpha* and occasionally undergoing heavy changes.

### How

* If you want to report a bug please attach the *very verbose* (`pdfcpu cmd -vv ...`) output and ideally a test PDF that you can share.
* Always make sure your contribution is based on the latest commit.
* Please sign your commits.

### Reporting Crashes

Unfortunately crashes do happen :(
For the majority of the cases this is due to a diverse pool of PDF Writers out there and millions of PDF files using different versions waiting to be processed by pdfcpu. Sometimes these PDFs were written more than 20(!) years ago. Often there is an issue with validation - sometimes a bug in the parser. Many times even using relaxed validation with pdfcpu does not work. In these cases we need to extend relaxed validation and for this we are relying on your help. By reporting crashes you are helping to improve the stability of pdfcpu. If you happen to crash on any pdfcpu operation be it on the command line or in your Go backend these are the steps to report this:

Regardless of the pdfcpu operation, please start using the pdfcpu command line to validate your file:

``` sh
pdfcpu validate -v &> crash.log
```

 or to produce very verbose output

 ``` sh
 pdfcpu validate -vv &> crash.log
 ```

will produce what's needed to investigate a crash. Then open an issue and post `crash.log` or its contents. Ideally post a test PDF you can share to reproduce this. You can also email to hhrutter@gmail.com or if you prefer Slack you can get in touch on the Gopher slack #pdfcpu channel.

If processing your PDF with pdfcpu crashes during validation and can be opened by Adobe Reader and Mac Preview chances are we can extend relaxed validation and provide a fix. If the file in question cannot be opened by both Adobe Reader and Mac Preview we cannot help you!

## Contributors

Thanks goes to these wonderful people:
<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
||||||||
| :---: | :---: | :---: | :---: | :---: |  :---: | :---: |
| [<img src="https://avatars1.githubusercontent.com/u/11322155?v=4" width="100px"/><br /><sub><b>Horst Rutter</b></sub>](https://github.com/hhrutter)<br /> |[<img src="https://avatars0.githubusercontent.com/u/5140211?v=4" width="100px"/><br /><sub><b>haldyr</b></sub>](https://github.com/haldyr)<br /> | [<img src="https://avatars3.githubusercontent.com/u/20608155?v=4" width="100px"/><br /><sub><b>Vyacheslav</b></sub>](https://github.com/SimePel)<br /> | [<img src="https://avatars1.githubusercontent.com/u/617459?v=4" width="100px"/><br /><sub><b>Erik Unger</b></sub>](https://github.com/ungerik)<br /> | [<img src="https://avatars1.githubusercontent.com/u/13079058?v=4" width="100px"/><br /><sub><b>Richard Wilkes</b></sub>](https://github.com/richardwilkes)<br /> | [<img src="https://avatars1.githubusercontent.com/u/16303386?s=400&v=4" width="100px"/><br /><sub><b>minenok-tutu</b></sub>](https://github.com/minenok-tutu)<br /> | [<img src="https://avatars0.githubusercontent.com/u/1965445?s=400&v=4" width="100px"/><br /><sub><b>Mateusz Burniak</b></sub>](https://github.com/matbur)<br /> |
| [<img src="https://avatars2.githubusercontent.com/u/1175110?s=400&v=4" width="100px"/><br /><sub><b>Dmitry Harnitski</b></sub>](https://github.com/dharnitski)<br /> |[<img src="https://avatars0.githubusercontent.com/u/1074083?s=400&v=4" width="100px"/><br /><sub><b>ryarnyah</b></sub>](https://github.com/ryarnyah)<br /> |[<img src="https://avatars0.githubusercontent.com/u/13267?s=400&v=4" width="100px"/><br /><sub><b>Sam Giffney</b></sub>](https://github.com/s01ipsist)<br /> |[<img src="https://avatars3.githubusercontent.com/u/32948066?s=400&v=4" width="100px"/><br /><sub><b>Carlos Eduardo Witte</b></sub>](https://github.com/cewitte)<br /> 
<!-- ALL-CONTRIBUTORS-LIST:END - Do not remove or modify this section -->

## Code of Conduct

Please note that this project is released with a Contributor [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Disclaimer

Usage of pdfcpu assumes you know about and respect all copyrights of any PDF content you may be processing. This applies to the PDF files as such, their content and in particular all embedded resources like font files or images. Credit goes to [Renee French](https://instagram.com/reneefrench) for creating our beloved Gopher.

## License

Apache-2.0

## Powered By

<p align="center">
  <a href="https://golang.org"> <img src="resources/Go-Logo_Aqua.png" width="200"> </a>
</p>
