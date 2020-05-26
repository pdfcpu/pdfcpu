---
layout: default
---

# Installation


## Download
Get the latest binary [here](https://github.com/pdfcpu/pdfcpu/releases).


## Using GOPATH

Required go version for building: go1.14 and up

```
go get github.com/pdfcpu/pdfcpu/cmd/...
cd $GOPATH/src/github.com/pdfcpu/pdfcpu/cmd/pdfcpu
go install
pdfcpu version
```

## Using Go Modules

```
git clone https://github.com/pdfcpu/pdfcpu
cd pdfcpu/cmd/pdfcpu
go install
pdfcpu version
```

## Using Homebrew (macOS)
```
brew install pdfcpu
```

## Run pdfcpu in a Docker container

```
docker build -t pdfcpu .
# mount current folder into container to process local files
docker run -it --mount type=bind,source="$(pwd)",target=/app pdfcpu ./pdfcpu validate -mode strict /app/pdfs/a.pdf
```
