# pdfcpu: a PDF processor written in Go with CLI and API support

[![Test](https://github.com/pdfcpu/pdfcpu/workflows/Test/badge.svg)](https://github.com/pdfcpu/pdfcpu/actions)
[![Coverage Status](https://coveralls.io/repos/github/pdfcpu/pdfcpu/badge.svg?branch=master)](https://coveralls.io/github/pdfcpu/pdfcpu?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/pdfcpu/pdfcpu.svg)](https://pkg.go.dev/github.com/pdfcpu/pdfcpu)
[![Go Report Card](https://goreportcard.com/badge/github.com/pdfcpu/pdfcpu)](https://goreportcard.com/report/github.com/pdfcpu/pdfcpu)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Sponsor](https://img.shields.io/badge/Sponsor-GitHub-%23fe8e86?logo=githubsponsors)](https://github.com/sponsors/hhrutter)

<p align="left">
  <a href="https://pdfcpu.io"><img src="resources/logoSmall.png" width="150"></a>
  <a href="https://pdfa.org"><img src="resources/pdfa.png" width="75"></a>
</p>

pdfcpu is a PDF processing library written in Go.

It is compatible with all PDF versions. Support for PDF 2.0 (ISO-32000-2) is evolving and continuously improving.

---

## Installation

### CLI

Prebuilt binaries and installation instructions (Homebrew, Docker, packages):

[Installation instructions](https://pdfcpu.io/install/cli)


### Go API

Install pdfcpu as a Go module:

```
go get github.com/pdfcpu/pdfcpu
```

API documentation:

[Installation and usage](https://pdfcpu.io/install/api)


---

## Usage

### CLI

Validate against PDF 2.0 (ISO-32000-2):
```
pdfcpu validate input.pdf
```

Merge two PDFs:
```
pdfcpu merge out.pdf in1.pdf in2.pdf
```

### Go API

See API documentation for usage examples.

---

## Features

* Validate, optimize, split, trim, and merge PDFs
* Extract and manipulate images, fonts, and metadata
* Encrypt and decrypt PDFs
* Resize and rotate pages
* Add and remove stamps and watermarks
* Manage digital signatures (ongoing work)
* Manage attachments and more...

## In Action

Common operations and examples:

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

---

## Command Set

Complete list of supported commands:

| | | |
|---|---|---|
| [annotations](https://pdfcpu.io/annot/annot) | [attachments](https://pdfcpu.io/attach/attach) | [booklet](https://pdfcpu.io/generate/booklet) |
| [bookmarks](https://pdfcpu.io/bookmarks/bookmarks) | [boxes](https://pdfcpu.io/boxes/boxes) | [certificates](https://pdfcpu.io/core/certs) |
| [change owner password](https://pdfcpu.io/encrypt/change_opw) | [change user password](https://pdfcpu.io/encrypt/change_upw) | [collect](https://pdfcpu.io/core/collect) |
| [config](https://pdfcpu.io/config/config) | [create](https://pdfcpu.io/create/create) | [crop](https://pdfcpu.io/core/crop) |
| [cut](https://pdfcpu.io/generate/cut) | [decrypt](https://pdfcpu.io/encrypt/decryptPDF) | [encrypt](https://pdfcpu.io/encrypt/encryptPDF) |
| [extract](https://pdfcpu.io/extract/extract) | [fonts](https://pdfcpu.io/fonts/fonts) | [form](https://pdfcpu.io/form/form) |
| [grid](https://pdfcpu.io/generate/grid) | [images](https://pdfcpu.io/images/images) | [import](https://pdfcpu.io/generate/import) |
| [info](https://pdfcpu.io/info) | [keywords](https://pdfcpu.io/keywords/keywords) | [merge](https://pdfcpu.io/core/merge) |
| [ndown](https://pdfcpu.io/generate/ndown) | [nup](https://pdfcpu.io/generate/nup) | [optimize](https://pdfcpu.io/core/optimize) |
| [pagelayout](https://pdfcpu.io/pagelayout/pagelayout) | [pagemode](https://pdfcpu.io/pagemode/pagemode) | [pages](https://pdfcpu.io/pages/pages) |
| [permissions](https://pdfcpu.io/encrypt/perm_set) | [portfolio](https://pdfcpu.io/portfolio/portfolio) | [poster](https://pdfcpu.io/generate/poster) |
| [properties](https://pdfcpu.io/properties/properties) | [resize](https://pdfcpu.io/core/resize) | [rotate](https://pdfcpu.io/core/rotate) |
| [signatures](https://pdfcpu.io/core/sign) | [split](https://pdfcpu.io/core/split) | [stamp](https://pdfcpu.io/core/stamp) |
| [trim](https://pdfcpu.io/core/trim) | [validate](https://pdfcpu.io/core/validate) | [viewerpref](https://pdfcpu.io/viewerpref/viewerpref) |
| [watermark](https://pdfcpu.io/core/watermark) | [zoom](https://pdfcpu.io/core/zoom) | |

---

## Motivation

pdfcpu aims to provide comprehensive PDF processing capabilities implemented in Go.

It focuses on correctness, robustness and independence from external dependencies.

---

## Focus

- comprehensive PDF processing functionality  
- minimal external dependencies  
- predictable and stable behavior  

---

## Documentation

* Project documentation: https://pdfcpu.io

### CLI

* Command help: `pdfcpu [command] --help`

### Go API

* Package documentation: https://pkg.go.dev/github.com/pdfcpu/pdfcpu
* API documentation: https://pkg.go.dev/github.com/pdfcpu/pdfcpu/pkg/api
* Examples:

  * https://github.com/pdfcpu/pdfcpu/tree/master/pkg/api/test
  * https://github.com/pdfcpu/pdfcpu/tree/master/pkg/samples

---

## Contributing

Contributions are welcome.

* Report bugs or propose changes via [issues](https://github.com/pdfcpu/pdfcpu/issues/new/choose)
* Discuss ideas on the [discussion board](https://github.com/pdfcpu/pdfcpu/discussions)
* For PRs, please open an issue or discussion first
v
### Guidelines

* Base your work on the latest commit
* Include verbose output (`pdfcpu cmd -vv ...`) and a sample PDF when reporting issues
* Please sign your commits

### Reporting crashes

Crashes may occur due to the wide variety of PDF producers and formats in use, including older or non-compliant files. In many cases this is related to validation issues or edge cases in the parser.

Even with relaxed validation, some files cannot be processed. These cases are essential for improving pdfcpu by extending validation and handling additional real-world PDFs.

If you encounter a crash, please report it.

Start by validating the file using the CLI:

```bash
pdfcpu validate -vv <file.pdf>
```

Include in your report:

* the command used
* verbose output (`-vv`)
* a sample PDF (if possible)

If validation crashes for a PDF that opens in Adobe Reader or macOS Preview, it is likely we can extend relaxed validation and provide a fix.

If the file cannot be opened by both Adobe Reader and macOS Preview, we cannot support it.

Please include a sample PDF to reproduce the issue whenever possible.

---

## Contributors

Thanks 💚 to all contributors:

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
[<img src="https://avatars.githubusercontent.com/u/9918222?v=4" width="100px"/><br/><sub><b>LNAhri</b></sub>](https://github.com/LNAhri) | [<img src="https://avatars.githubusercontent.com/u/142796877?v=4" width="100px"/><br/><sub><b>vishal</b></sub>](https://github.com/vishal-at) | [<img src="https://avatars.githubusercontent.com/u/18169566?v=4" width="100px"/><br/><sub><b>Andreas Deininger</b></sub>](https://github.com/deining) | [<img src="https://avatars.githubusercontent.com/u/5825735?v=4" width="100px"/><br/><sub><b>Robert Raines</b></sub>](https://github.com/solintllc-robert) | [<img src="https://avatars.githubusercontent.com/u/316176?v=4" width="100px"/><br/><sub><b>Frank Anderson</b></sub>](https://github.com/frob) |  [<img src="https://avatars.githubusercontent.com/u/20972350?v=4" width="100px"/><br/><sub><b>Sven Lilienthal</b></sub>](https://github.com/SveLil) |  [<img src="https://avatars.githubusercontent.com/u/1900106?v=4" width="100px"/><br/><sub><b>Florian Kinder</b></sub>](https://github.com/fank) |
[<img src="https://avatars.githubusercontent.com/u/113192632?v=4" width="100px"/><br/><sub><b>mdmcconnell</b></sub>](https://github.com/mdmcconnell) | [<img src="https://avatars.githubusercontent.com/u/195061?v=4" width="100px"/><br/><sub><b>Bradley Erickson</b></sub>](https://github.com/13rac1) | [<img src="https://avatars.githubusercontent.com/u/10998835?v=4" width="100px"/><br/><sub><b>doronbehar</b></sub>](https://github.com/doronbehar) |[<img src="https://avatars.githubusercontent.com/u/48005710?v=4" width="100px"/><br/><sub><b>joeyave</b></sub>](https://github.com/joeyave) ||||

<!-- ALL-CONTRIBUTORS-LIST:END - Do not remove or modify this section -->

---

## Code of Conduct

This project is released with a Contributor [Code of Conduct](CODE_OF_CONDUCT.md).
By participating, you agree to abide by its terms.

---

## Disclaimer

Use of pdfcpu assumes compliance with all applicable copyrights for any processed PDF content, including embedded resources such as fonts and images.

Gopher artwork by [Renee French](https://www.instagram.com/reneefrenchphoto)

---

## License

Apache License 2.0

---

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=592f747a-ac99-42e3-802a-80dbf2b94519&page=README.md" />
