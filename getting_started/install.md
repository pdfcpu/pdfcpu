---
layout: default
---

# Installation


## Download
Get the latest binary [here](https://github.com/pdfcpu/pdfcpu/releases).

Required go version for building: go1.20 and up


## Using Go Modules

```
$ git clone https://github.com/pdfcpu/pdfcpu
$ cd pdfcpu/cmd/pdfcpu
$ go install
$ pdfcpu version
```

## Using Homebrew (macOS)
```
$ brew install pdfcpu
$ pdfcpu version
```
## Using DNF/YUM (Fedora)
```
$ sudo dnf install golang-github-pdfcpu
$ pdfcpu version
```

## Run in a Docker container

```
$ docker build -t pdfcpu .
# mount current folder into container to process local files
$ docker run -it --mount type=bind,source="$(pwd)",target=/app pdfcpu ./pdfcpu validate /app/pdfs/a.pdf
```
