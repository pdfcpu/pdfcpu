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

## Using Go install:
```
$ go install github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest
```

## Using Homebrew (macOS)
```
$ brew install pdfcpu
$ pdfcpu version

```
## Using DNF/YUM (Fedora)
(maybe not current)
```
$ sudo dnf install golang-github-pdfcpu
$ pdfcpu version
```

## Run in a Docker container
```
$ docker build -t pdfcpu .
# mount current host folder into container as /app to process files in the local host folder
$ docker run -it -v "$(pwd)":/app pdfcpu validate a.pdf
```
