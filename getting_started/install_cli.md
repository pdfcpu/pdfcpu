---
layout: default
---

# CLI Installation

Download a prebuilt binary for your platform and run `pdfcpu version` to verify installation.

---

## Download

### macOS

* [Apple Silicon (arm64)](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Darwin_arm64.tar.xz)
* [Intel (x86_64)](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Darwin_x86_64.tar.xz)

### Linux

* [x86_64](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Linux_x86_64.tar.xz)
* [arm64](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Linux_arm64.tar.xz)
* [armv7](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Linux_armv7.tar.xz)
* [i386](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Linux_i386.tar.xz)

### Windows

* [x86_64](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Windows_x86_64.zip)
* [i386](https://dl.pdfcpu.io/releases/download/v0.12.0/pdfcpu_0.12.0_Windows_i386.zip)

### Checksums

* [checksums.txt](https://dl.pdfcpu.io/releases/download/v0.12.0/checksums.txt)

---

## Install

1. Extract the archive.
2. Run:

    pdfcpu version

3. Optional: move the binary to a directory in your `PATH`:

    sudo mv pdfcpu /usr/local/bin/

---

## Using Go

Install the CLI tool:

    go install github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest
    pdfcpu version

---

## Package Managers

### Homebrew (macOS)

    brew install pdfcpu
    pdfcpu version

### MacPorts

    sudo port install pdfcpu
    pdfcpu version

### DNF/YUM (Fedora)

    sudo dnf install golang-github-pdfcpu
    pdfcpu version

---

## Docker

    docker build -t pdfcpu .
    docker run -it -v "$(pwd)":/app pdfcpu validate a.pdf

---

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=0203eab5-b03d-4fd2-b2f1-2c505e09cbe2" />