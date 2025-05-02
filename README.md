# pdfcpu: a Go PDF processor and CLI

[![Test](https://github.com/pdfcpu/pdfcpu/workflows/Test/badge.svg)](https://github.com/pdfcpu/pdfcpu/actions)
[![Coverage Status](https://coveralls.io/repos/github/pdfcpu/pdfcpu/badge.svg?branch=master)](https://coveralls.io/github/pdfcpu/pdfcpu?branch=master)
[![GoDoc](https://godoc.org/github.com/pdfcpu/pdfcpu?status.svg)](https://pkg.go.dev/github.com/pdfcpu/pdfcpu)
[![Go Report Card](https://goreportcard.com/badge/github.com/pdfcpu/pdfcpu)](https://goreportcard.com/report/github.com/pdfcpu/pdfcpu)
[![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://opensource.org/licenses/Apache-2.0)
[![Gurubase](https://img.shields.io/badge/Gurubase-Ask%20pdfcpu%20Guru-006BFF)](https://gurubase.io/g/pdfcpu)

[![Latest release](https://img.shields.io/github/release/pdfcpu/pdfcpu.svg)](https://github.com/pdfcpu/pdfcpu/releases)
[![](https://img.shields.io/static/v1?label=Sponsor&message=%E2%9D%A4&logo=GitHub&color=%23fe8e86)](https://github.com/sponsors/hhrutter)


<a href="https://pdfcpu.io"><img src="resources/logoSmall.png" width="150"></a>
<a href="https://pdfa.org"><img src="resources/pdfa.png" width="75"></a>

pdfcpu is a PDF processing library written in [Go](https://go.dev/) that supports encryption and offers both an API and a command-line interface (CLI). It is compatible with all PDF versions with basic support and ongoing improvement for PDF 2.0 (ISO-32000-2).


## Motivation

This is an effort to build a comprehensive PDF processing library from the ground up written in Go. Over time pdfcpu aims to support the standard range of PDF processing features and also any interesting use cases that may present themselves along the way.

<p align="center">
  <kbd><a href="https://pdfcpu.io/generate/grid"><img src="resources/gridpdf.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/watermark"><img src="resources/wmi1abs.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/generate/nup"><img src="resources/nup9pdf.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/fonts/fonts"><img src="resources/cjkv.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/4exp.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/form/form"><img src="resources/form.png" height="150"></a></kbd><br><br>
  <kbd><a href="https://pdfcpu.io/generate/create"><img src="resources/table.png" height="100"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/sti.png" height="150"></a></kbd>&nbsp;
  <kbd><img src="resources/hold3.png" height="150"></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/core/watermark"><img src="resources/wmi4.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/generate/create"><img src="resources/imagebox.png" height="100"></a></kbd>&nbsp;<br><br>
  <kbd><a href="https://pdfcpu.io/generate/booklet"><img src="resources/book2A4p1.png" height="150"></a></kbd>
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/stp.png" height="150"></a></kbd>&nbsp;
  <kbd><a href="https://pdfcpu.io/generate/grid"><img src="resources/gridimg.png" height="150"></a></kbd>
  <kbd><a href="https://pdfcpu.io/core/stamp"><img src="resources/stRoundBorder.png" height="150"></a></kbd>
  <kbd><a href="https://pdfcpu.io/generate/booklet"><img src="resources/book4A4p1.png" height="150"></a></kbd>
</p>

## Focus

The primary emphasis is on providing robust assistance for batch processing and scripting through a comprehensive command-line interface.
Simultaneously, pdfcpu aims to simplify the integration of PDF processing into your Go-based backend system by offering a versatile set of commands.

## Command Set

* [annotations](https://pdfcpu.io/annot/annot)
* [attachments](https://pdfcpu.io/attach/attach)
* [booklet](https://pdfcpu.io/generate/booklet)
* [bookmarks](https://pdfcpu.io/bookmarks/bookmarks)
* [boxes](https://pdfcpu.io/boxes/boxes)
* [certificates](https://pdfcpu.io/core/certs)
* [change owner password](https://pdfcpu.io/encrypt/change_opw)
* [change user password](https://pdfcpu.io/encrypt/change_upw)
* [collect](https://pdfcpu.io/core/collect)
* [config](https://pdfcpu.io/config/config)
* [create](https://pdfcpu.io/create/create)
* [crop](https://pdfcpu.io/core/crop)
* [cut](https://pdfcpu.io/generate/cut)
* [decrypt](https://pdfcpu.io/encrypt/decryptPDF)
* [encrypt](https://pdfcpu.io/encrypt/encryptPDF)
* [extract](https://pdfcpu.io/extract/extract)
* [fonts](https://pdfcpu.io/fonts/fonts)
* [form](https://pdfcpu.io/form/form)
* [grid](https://pdfcpu.io/generate/grid)
* [images](https://pdfcpu.io/images/images)
* [import](https://pdfcpu.io/generate/import)
* [info](https://pdfcpu.io/info)
* [keywords](https://pdfcpu.io/keywords/keywords)
* [merge](https://pdfcpu.io/core/merge)
* [ndown](https://pdfcpu.io/generate/ndown)
* [nup](https://pdfcpu.io/generate/nup)
* [optimize](https://pdfcpu.io/core/optimize)
* [pagelayout](https://pdfcpu.io/pagelayout/pagelayout)
* [pagemode](https://pdfcpu.io/pagemode/pagemode)
* [pages](https://pdfcpu.io/pages/pages)
* [permissions](https://pdfcpu.io/encrypt/perm_set)
* [portfolio](https://pdfcpu.io/portfolio/portfolio)
* [poster](https://pdfcpu.io/generate/poster)
* [properties](https://pdfcpu.io/properties/properties)
* [resize](https://pdfcpu.io/core/resize)
* [rotate](https://pdfcpu.io/core/rotate)
* [signatures](http://pdfcpu.io/core/sign)
* [split](https://pdfcpu.io/core/split)
* [stamp](https://pdfcpu.io/core/stamp)
* [trim](https://pdfcpu.io/core/trim)
* [validate](https://pdfcpu.io/core/validate)
* [viewerpref](https://pdfcpu.io/viewerpref/viewerpref)
* [watermark](https://pdfcpu.io/core/watermark)
* [zoom](https://pdfcpu.io/core/zoom)

## Documentation

* [pdfcpu.io](https://pdfcpu.io)
* [API tests](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/api/test)
* [API samples](https://github.com/pdfcpu/pdfcpu/tree/master/pkg/samples) 
* CLI usage: `$ pdfcpu help cmd`

### GoDoc

* [pdfcpu package](https://pkg.go.dev/github.com/pdfcpu/pdfcpu)
* [pdfcpu API](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api)
* [pdfcpu CLI](https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/cli)

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


### Using Go Modules

```
$ git clone https://github.com/pdfcpu/pdfcpu
$ cd pdfcpu/cmd/pdfcpu
$ go install
$ pdfcpu version
```
or directly through Go install:

```
$ go install github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest
```

### Using Homebrew (macOS)
```
$ brew install pdfcpu
$ pdfcpu version
```

### Using DNF/YUM (Fedora)
```
$ sudo dnf install golang-github-pdfcpu
$ pdfcpu version
```

### Run in a Docker container

```shell
$ docker build -t pdfcpu .
# mount current host folder into container as /app to process files in the local host folder
$ docker run -it -v "$(pwd)":/app pdfcpu validate a.pdf
```

## Contributing

### What

* Please [create](https://github.com/pdfcpu/pdfcpu/issues/new/choose) an issue if you find a bug or want to propose a change.
* Feature requests - always welcome!
* Bug fixes - always welcome!
* PRs - let's [discuss](https://github.com/pdfcpu/pdfcpu/discussions) first or [create](https://github.com/pdfcpu/pdfcpu/issues/new/choose) an issue.
* pdfcpu is stable but still *Alpha* and occasionally undergoing heavy changes.

### How

* The pdfcpu [discussion board](https://github.com/pdfcpu/pdfcpu/discussions) is open! Please engage in any form helpful for the community.
* If you want to report a bug please attach the *very verbose* (`pdfcpu cmd -vv ...`) output and ideally a test PDF that you can share.
* Always make sure your contribution is based on the latest commit.
* Please sign your commits.

### Reporting Crashes

Unfortunately crashes do happen :(
For the majority of the cases this is due to a diverse pool of PDF Writers out there and millions of PDF files using different versions waiting to be processed by pdfcpu. Sometimes these PDFs were written more than 20(!) years ago. Often there is an issue with validation - sometimes a bug in the parser. Many times even using relaxed validation with pdfcpu does not work. In these cases we need to extend relaxed validation and for this we are relying on your help. By reporting crashes you are helping to improve the stability of pdfcpu. If you happen to crash on any pdfcpu operation be it on the command line or in your Go backend these are the steps to report this:

Regardless of the pdfcpu operation, please start using the pdfcpu command line to validate your file:

``` sh
$ pdfcpu validate -v &> crash.log
```

 or to produce very verbose output

 ``` sh
 $ pdfcpu validate -vv &> crash.log
 ```

will produce what's needed to investigate a crash. Then open an issue and post `crash.log` or its contents. Ideally post a test PDF you can share to reproduce this. You can also email to hhrutter@gmail.com or if you prefer Slack you can get in touch on the Gopher slack #pdfcpu channel.

If processing your PDF with pdfcpu crashes during validation and can be opened by Adobe Reader and Mac Preview chances are we can extend relaxed validation and provide a fix. If the file in question cannot be opened by both Adobe Reader and Mac Preview we cannot help you!

## Contributors

Thanks 💚 goes to these wonderful people:

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
||||||||
| :---: | :---: | :---: | :---: | :---: |  :---: | :---: |
| [<img src="https://avatars1.githubusercontent.com/u/11322155?v=4" width="100px"/><br/><sub><b>Horst Rutter</b></sub>](https://github.com/hhrutter) | [<img src="https://avatars0.githubusercontent.com/u/5140211?v=4" width="100px"/><br/><sub><b>haldyr</b></sub>](https://github.com/haldyr) | [<img src="https://avatars3.githubusercontent.com/u/20608155?v=4" width="100px"/><br/><sub><b>Vyacheslav</b></sub>](https://github.com/SimePel) | [<img src="https://avatars1.githubusercontent.com/u/617459?v=4" width="100px"/><br/><sub><b>Erik Unger</b></sub>](https://github.com/ungerik) | [<img src="https://avatars1.githubusercontent.com/u/13079058?v=4" width="100px"/><br/><sub><b>Richard Wilkes</b></sub>](https://github.com/richardwilkes) | [<img src="https://avatars1.githubusercontent.com/u/16303386?s=400&v=4" width="100px"/><br/><sub><b>minenok-tutu</b></sub>](https://github.com/minenok-tutu) | [<img src="https://avatars0.githubusercontent.com/u/1965445?s=400&v=4" width="100px"/><br/><sub><b>Mateusz Burniak</b></sub>](https://github.com/matbur) |
| [<img src="https://avatars2.githubusercontent.com/u/1175110?s=400&v=4" width="100px"/><br/><sub><b>Dmitry Harnitski</b></sub>](https://github.com/dharnitski) | [<img src="https://avatars0.githubusercontent.com/u/1074083?s=400&v=4" width="100px"/><br/><sub><b>ryarnyah</b></sub>](https://github.com/ryarnyah) | [<img src="https://avatars0.githubusercontent.com/u/13267?s=400&v=4" width="100px"/><br/><sub><b>Sam Giffney</b></sub>](https://github.com/s01ipsist) | [<img src="https://avatars3.githubusercontent.com/u/32948066?s=400&v=4" width="100px"/><br /><sub><b>Carlos Eduardo Witte</b></sub>](https://github.com/cewitte) | [<img src="https://avatars1.githubusercontent.com/u/2374948?s=400&u=a36e5f8da8dc1c102bc4d283f25e4c61cae7f985&v=4" width="100px"/><br/><sub><b>minusworld</b></sub>](https://github.com/minusworld) | [<img src="https://avatars0.githubusercontent.com/u/18538487?s=400&u=b9e628dfc60f672a887be2ed04a791195829943e&v=4" width="100px"/><br/><sub><b>Witold Konior</b></sub>](https://github.com/jozuenoon) | [<img src="https://avatars0.githubusercontent.com/u/630151?s=400&v=4" width="100px"/><br/><sub><b>joonas.fi</b></sub>](https://github.com/joonas-fi) |
| [<img src="https://avatars3.githubusercontent.com/u/10349817?s=400&u=93bacb23bd2909d5b6c5b644a8d4cdd947422ee1&v=4" width="100px"/><br/><sub><b>Henrik Reinstädtler</b></sub>](https://github.com/henrixapp) | [<img src="https://avatars1.githubusercontent.com/u/72016286?s=400&v=4" width="100px"/><br/><sub><b>VMorozov-wh</b></sub>](https://github.com/VMorozov-wh) | [<img src="https://avatars0.githubusercontent.com/u/31929422?s=400&v=4" width="100px"/><br/><sub><b>Benoit KUGLER</b></sub>](https://github.com/benoitkugler) | [<img src="https://avatars.githubusercontent.com/u/704919?s=400&v=4" width="100px"/><br/><sub><b>Adam Greenhall</b></sub>](https://github.com/adamgreenhall) | [<img src="https://avatars.githubusercontent.com/u/5201812?s=400&u=8a0a9fca4560be71d4923299ddebf877854eea54&v=4" width="100px"/><br/><sub><b>moritamori</b></sub>](https://github.com/moritamori) | [<img src="https://avatars.githubusercontent.com/u/41904529?s=400&u=044396494285ad806e86d1936c390b3071ce57c0&v=4" width="100px"/><br/><sub><b>JanBaryla</b></sub>](https://github.com/JanBaryla) | [<img src="https://avatars.githubusercontent.com/u/43145244?s=400&u=89a689f1a854ce0f57ae2a0333c82bfdc5723bb9&v=4" width="100px"/><br/><sub><b>TheDiscordian</b></sub>](https://github.com/TheDiscordian) |
| [<img src="https://avatars.githubusercontent.com/u/15472552?v=4" width="100px"/><br/><sub><b>Rafael Garcia Argente</b></sub>](https://github.com/rgargente) | [<img src="https://avatars.githubusercontent.com/u/710057?v=4" width="100px"/><br/><sub><b>truyet</b></sub>](https://github.com/truyet) | [<img src="https://avatars.githubusercontent.com/u/5031217?v=4" width="100px"/><br/><sub><b>Christian Nicola</b></sub>](https://github.com/christiannicola) | [<img src="https://avatars.githubusercontent.com/u/3233970?v=4" width="100px"/><br/><sub><b>Benjamin Krill</b></sub>](https://github.com/kben) | [<img src="https://avatars.githubusercontent.com/u/26521615?v=4" width="100px"/><br/><sub><b>Peter Wyatt</b></sub>](https://github.com/petervwyatt) | [<img src="https://avatars.githubusercontent.com/u/3142701?v=4" width="100px"/><br/><sub><b>Kroum Tzanev</b></sub>](https://github.com/kpym) | [<img src="https://avatars.githubusercontent.com/u/992878?v=4" width="100px"/><br/><sub><b>Stefan Huber</b></sub>](https://github.com/signalwerk) |
| [<img src="https://avatars.githubusercontent.com/u/59667587?v=4" width="100px"/><br/><sub><b>Juan Iscar</b></sub>](https://github.com/juaismar) | [<img src="https://avatars.githubusercontent.com/u/20135478?v=4" width="100px"/><br/><sub><b>Eng Zer Jun</b></sub>](https://github.com/Juneezee) | [<img src="https://avatars.githubusercontent.com/u/28459131?v=4" width="100px"/><br/><sub><b>Dmitry Ivanov</b></sub>](https://github.com/hant0508)|[<img src="https://avatars.githubusercontent.com/u/16866547?v=4" width="100px"/><br/><sub><b>Rene Kaufmann</b></sub>](https://github.com/HeavyHorst)|[<img src="https://avatars.githubusercontent.com/u/26827864?v=4" width="100px"/><br/><sub><b>Christian Heusel</b></sub>](https://github.com/christian-heusel) | [<img src="https://avatars.githubusercontent.com/u/305673?v=4" width="100px"/><br/><sub><b>Chris</b></sub>](https://github.com/freshteapot) | [<img src="https://avatars.githubusercontent.com/u/2892794?v=4" width="100px"/><br/><sub><b>Lukasz Czaplinski</b></sub>](https://github.com/scoiatael) |
[<img src="https://avatars.githubusercontent.com/u/49206635?v=4" width="100px"/><br/><sub><b>Joel Silva Schutz</b></sub>](https://github.com/joelschutz) | [<img src="https://avatars.githubusercontent.com/u/28219076?v=4" width="100px"/><br/><sub><b>semvis123</b></sub>](https://github.com/semvis123) | [<img src="https://avatars.githubusercontent.com/u/8717479?v=4"  width="100px"/><br/><sub><b>guangwu</b></sub>](https://github.com/testwill) | [<img src="https://avatars.githubusercontent.com/u/4014912?v=4"  width="100px"/><br/><sub><b>Yoshiki Nakagawa</b></sub>](https://github.com/yyoshiki41) | [<img src="https://avatars.githubusercontent.com/u/432860?v=4"  width="100px"/><br/><sub><b>Steve van Loben Sels</b></sub>](https://github.com/stevevls) | [<img src="https://avatars.githubusercontent.com/u/6083533?v=4" width="100px"/><br/><sub><b>Yaofu</b></sub>](https://github.com/mygityf) | [<img src="https://avatars.githubusercontent.com/u/15724278?v=4" width="100px"/><br/><sub><b>vsenko</b></sub>](https://github.com/vsenko) |
[<img src="https://avatars.githubusercontent.com/u/16507?v=4" width="100px"/><br/><sub><b>Alexis Hildebrandt</b></sub>](https://github.com/afh) | [<img src="https://avatars.githubusercontent.com/u/1395040?v=4" width="100px"/><br/><sub><b>Sivukhin Nikita</b></sub>](https://github.com/sivukhin)  | [<img src="https://avatars.githubusercontent.com/u/247730?v=4"  width="100px"/><br/><sub><b>Joachim Bauch</b></sub>](https://github.com/fancycode) | [<img src="https://avatars.githubusercontent.com/u/127291996?v=4"  width="100px"/><br/><sub><b>kalimit</b></sub>](https://github.com/kalimit) | [<img src="https://avatars.githubusercontent.com/u/5080535?v=4"  width="100px"/><br/><sub><b>Andreas Erhard</b></sub>](https://github.com/xelan) | [<img src="https://avatars.githubusercontent.com/u/32378535?v=4"  width="100px"/><br/><sub><b>Matsumoto Toshi</b></sub>](https://github.com/toshi1127) | [<img src="https://avatars.githubusercontent.com/u/440634?v=4"  width="100px"/><br/><sub><b>Carl Wilson</b></sub>](https://github.com/carlwilson) |
[<img src="https://avatars.githubusercontent.com/u/9918222?v=4" width="100px"/><br/><sub><b>LNAhri</b></sub>](https://github.com/LNAhri) | [<img src="https://avatars.githubusercontent.com/u/142796877?v=4" width="100px"/><br/><sub><b>vishal</b></sub>](https://github.com/vishal-at) | [<img src="https://avatars.githubusercontent.com/u/18169566?v=4" width="100px"/><br/><sub><b>Andreas Deininger</b></sub>](https://github.com/deining) |  | | | 











<!-- ALL-CONTRIBUTORS-LIST:END - Do not remove or modify this section -->

## Code of Conduct

Please note that this project is released with a Contributor [Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project you agree to abide by its terms.

## Disclaimer

Usage of pdfcpu assumes you know about and respect all copyrights of any PDF content you may be processing. This applies to the PDF files as such, their content and in particular all embedded resources like font files or images. Credit goes to [Renee French](https://instagram.com/reneefrench) for creating our beloved Gopher.

## License

Apache-2.0

