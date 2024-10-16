---
layout: default
---

# Update Images

This command lets you replace individual images by object number or page number and resource id.

The necessary info is retrieved from the output of `pdfcpu images list`.

Have a look at some [examples](#examples).

## Usage

```
pdfcpu images update inFile imageFile [outFile] [ objNr | (pageNr Id) ]
````

<br>

### Common Flags

| name                                            | description     | values
|:------------------------------------------------|:----------------|:-------
| [v(erbose)](../getting_started/common_flags.md) | turn on logging |
| [vv](../getting_started/common_flags.md)        | verbose logging |
| [q(uiet)](../getting_started/common_flags.md)   | quiet mode      |
| [u(nit)](../getting_started/common_flags.md)    | display unit    | po(ints),in(ches),cm,mm
| [c(onf)](../getting_started/common_flags.md)       | config dir      | $path, disable
| [upw](../getting_started/common_flags.md)          | user password   |
| [opw](../getting_started/common_flags.md)          | owner password  |

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes
| objNr        | object number       | no
| pageNr       | page number         | no
| Id           | resource id         | no

<br>

## Examples

List all contained images:

```sh
$ pdfcpu images list gallery.pdf
gallery.pdf:
1 images available (1.8 MB)

Page Obj# │ Id  │ Type  SoftMask ImgMask │ Width │ Height │ ColorSpace Comp bpc Interp │   Size │ Filters
━━━━━━━━━━┿━━━━━┿━━━━━━━━━━━━━━━━━━━━━━━━┿━━━━━━━┿━━━━━━━━┿━━━━━━━━━━━━━━━━━━━━━━━━━━━━┿━━━━━━━━┿━━━━━━━━━━━━
   1    3 │ Im0 │ image                  │  1268 │    720 │  DeviceRGB    3   8    *   │ 1.8 MB │ FlateDecode
```

<br>

Extract all images into the current dir:

```sh
$ pdfcpu images extract gallery.pdf .
extracting images from gallery.pdf into ./ ...
optimizing...
writing gallery_1_Im0.png
```

<br>

Update image with Id=Im0 on page=1 with gallery_1_Im0.png and write the result to updatedGallery.pdf.<br>
Here page number and resource id are contained in the image file name:

```sh
$ pdfcpu images update gallery.pdf gallery_1_Im0.png updatedGallery.pdf
```

<br>

Update image with object number 3 with logo.png:

```sh
$ pdfcpu images update gallery.pdf logo.png 3
```

<br>

update image with Id=Im0 on page=1 with logo.jpg

```sh
$ pdfcpu images update gallery.pdf logo.jpg 1 Im0
```
