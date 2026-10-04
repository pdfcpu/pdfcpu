---
layout: default
title: "CLI Installation"
---

# CLI Installation

Download a prebuilt binary for your platform and run `pdfcpu version` to verify installation.

---

## Download

These downloads are for **v0.16.1**.
See [GitHub Releases](https://github.com/pdfcpu/pdfcpu/releases) for other versions.

### macOS

* [Apple Silicon (arm64)](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Darwin_arm64.tar.xz)
* [Intel (x86_64)](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Darwin_x86_64.tar.xz)

### Linux

* [x86_64](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Linux_x86_64.tar.xz)
* [arm64](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Linux_arm64.tar.xz)
* [armv7](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Linux_armv7.tar.xz)
* [i386](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Linux_i386.tar.xz)

### Windows

* [x86_64](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Windows_x86_64.zip)
* [i386](https://dl.pdfcpu.io/releases/download/v0.16.1/pdfcpu_0.16.1_Windows_i386.zip)

### Checksums

* [checksums.txt](https://dl.pdfcpu.io/releases/download/v0.16.1/checksums.txt)

---

### Software Bill of Materials (SBOMs)

Each download archive has a corresponding `.sbom.json` asset on the
[release page](https://github.com/pdfcpu/pdfcpu/releases/tag/v0.16.1). <br>
These JSON files describe application dependencies and are included
in `checksums.txt`.

The WebAssembly (`js/wasm`) SBOM is generated from the tagged source with
the WASM build settings. Its dependency inventory is source-derived rather
than extracted from the binary. <br>It includes SHA-256 hashes identifying the
WASM binary and its archive.

---

## Install

1. Extract the archive.
2. Run:

    pdfcpu version

3. Optional: move the binary to a directory in your `PATH`:

    sudo mv pdfcpu /usr/local/bin/

---

## Using Go

To install the latest stable release:

    go install github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest
    pdfcpu version

To build with the embedded EU Trusted List certificate bundles:

    go install -tags pdfcpu_eutl github.com/pdfcpu/pdfcpu/cmd/pdfcpu@latest
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

An official pdfcpu container image is planned for v0.17.0.

---

<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=0203eab5-b03d-4fd2-b2f1-2c505e09cbe2" width="1" height="1" />
