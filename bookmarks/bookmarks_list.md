---
layout: default
---

# List Bookmarks

This command prints a list of any existing bookmarks. 

Have a look at some [examples](#examples).

## Usage

```
pdfcpu bookmarks list inFile
```

<br>

### [Common Flags](../getting_started/common_flags)

<br>

### Arguments

| name         | description         | required
|:-------------|:--------------------|:--------
| inFile       | PDF input file      | yes

<br>

## Examples

 List all page boundaries of test.pdf:

```
$ pdfcpu bookmarks list bookmarkTree.pdf
Page 1: Level 1
    Page 2: Level 1.1
    Page 3: Level 1.2
        Page 4: Level 1.2.1
Page 5: Level 2
    Page 6: Level 2.1
    Page 7: Level 2.2
    Page 8: Level 2.3
```

<br>

You can also abbreviate the command like so:

```
$ pdfcpu bookm l bookmarkSimple.pdf
Page 1: Applicant’s Form
Page 2: Bold 这是一个测试
Page 3: Italic 测试 尾巴
Page 4: Bold & Italic
Page 16: The birthday of Smalltalk
Page 17: Gray
Page 18: Red
Page 19: Bold Red
```
