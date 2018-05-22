package main

const (
	usage = `pdfcpu is a tool for PDF manipulation written in Go. 
	
Usage:
	
	pdfcpu command [arguments]
		
The commands are:
	
	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)
	optimize	optimize PDF by getting rid of redundant page resources
	split		split multi-page PDF into several single-page PDFs
	merge		concatenate 2 or more PDFs
	extract		extract images, fonts, content or pages
	trim		create trimmed version
	attach		list, add, remove, extract embedded file attachments
	perm		list, add user access permissions
	encrypt		set password protection		
	decrypt		remove password protection
	changeupw	change user password
	changeopw	change owner password
	version		print version
   
	Single-letter Unix-style supported for commands and flags.

Use "pdfcpu help [command]" for more information about a command.`

	usageValidate     = "usage: pdfcpu validate [-verbose] [-mode strict|relaxed] [-upw userpw] [-opw ownerpw] inFile"
	usageLongValidate = `Validate checks inFile for specification compliance.

verbose ... extensive log output
   mode ... validation mode
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
		
The validation modes are:

 strict ... (default) validates against PDF 32000-1:2008 (PDF 1.7)
relaxed ... like strict but doesn't complain about common seen spec violations.`

	usageOptimize     = "usage: pdfcpu optimize [-verbose] [-stats csvFile] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongOptimize = `Optimize reads inFile, removes redundant page resources like embedded fonts and images and writes the result to outFile.

verbose ... extensive log output
  stats ... appends a stats line to a csv file with information about the usage of root and page entries.
            useful for batch optimization and debugging PDFs.
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
outFile ... output pdf file (default: inFile-new.pdf)`

	usageSplit     = "usage: pdfcpu split [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir"
	usageLongSplit = `Split generates a set of single page PDFs for the input file in outDir.

verbose ... extensive log output
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
 outDir ... output directory`

	usageMerge     = "usage: pdfcpu merge [-verbose] outFile inFile..."
	usageLongMerge = `Merge concatenates a sequence of PDFs/inFiles to outFile.

verbose ... extensive log output
outFile	... output pdf file
inFiles ... a list of at least 2 pdf files subject to concatenation.`

	usageExtract     = "usage: pdfcpu extract [-verbose] -mode image|font|content|page [-pages pageSelection] [-upw userpw] [-opw ownerpw] inFile outDir"
	usageLongExtract = `Extract exports inFile's images, fonts, content or pages into outDir.

verbose ... extensive log output
   mode ... extraction mode
  pages ... page selection
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
 outDir ... output directory

 The extraction modes are:

  image ... extract images (supported PDF filters: DCTDecode, JPXDecode)
   font ... extract font files (supported font types: TrueType)
content ... extract raw page content
   page ... extract single page PDFs`

	usageTrim     = "usage: pdfcpu trim [-verbose] -pages pageSelection [-upw userpw] [-opw ownerpw] inFile outFile"
	usageLongTrim = `Trim generates a trimmed version of inFile for selectedPages.

verbose ... extensive log output
  pages ... page selection
    upw ... user password
    opw ... owner password
 inFile ... input pdf file 
outFile ... output pdf file, the trimmed version of inFile`

	usagePageSelection = `pageSelection selects pages for processing and is a comma separated list of expressions:

Valid expressions are:

  # ... include page #               #-# ... include page range
 !# ... exclude page #              !#-# ... exclude page range
 n# ... exclude page #              n#-# ... exclude page range

 #- ... include page # - last page    -# ... include first page - page #
!#- ... exclude page # - last page   !-# ... exclude first page - page #
n#- ... exclude page # - last page   n-# ... exclude first page - page #

n serves as an alternative for !, since ! needs to be escaped with single quotes on the cmd line.

Valid pageSelections e.g. -3,5,7- or 4-7,!6 or 1-,!5

A missing pageSelection means all pages are selected for generation.`

	usageAttachList    = "pdfcpu attach list [-verbose] [-upw userpw] [-opw ownerpw] inFile"
	usageAttachAdd     = "pdfcpu attach add [-verbose] [-upw userpw] [-opw ownerpw] inFile file..."
	usageAttachRemove  = "pdfcpu attach remove [-verbose] [-upw userpw] [-opw ownerpw] inFile [file...]"
	usageAttachExtract = "pdfcpu attach extract [-verbose] [-upw userpw] [-opw ownerpw] inFile outDir [file...]"

	usageAttach = "usage: " + usageAttachList +
		"\n       " + usageAttachAdd +
		"\n       " + usageAttachRemove +
		"\n       " + usageAttachExtract

	usageLongAttach = `Attach manages embedded file attachments.
	
verbose ... extensive log output
   perm ... user access permissions
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
 outDir ... output directory`

	usagePermList = "pdfcpu perm list [-verbose] [-upw userpw] [-opw ownerpw] inFile"
	usagePermAdd  = "pdfcpu perm add [-verbose] [-perm none|all] [-upw userpw] -opw ownerpw inFile"

	usagePerm = "usage: " + usagePermList + "\n       " + usagePermAdd

	usageLongPerm = `Perm manages user access permissions.
	
verbose ... extensive log output
   perm ... user access permissions
    upw ... user password
    opw ... owner password
 inFile ... input pdf file`

	usageEncrypt     = "usage: pdfcpu encrypt [-verbose] [-mode rc4|aes] [-key 40|128] [perm none|all] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongEncrypt = `Encrypt sets a password protection based on user and owner password.

verbose ... extensive log output
   mode ... algorithm (default=aes)
    key ... key length in bits (default=128)
   perm ... user access permissions
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
outFile ... output pdf file`

	usageDecrypt     = "usage: pdfcpu decrypt [-verbose] [-upw userpw] [-opw ownerpw] inFile [outFile]"
	usageLongDecrypt = `Decrypt removes a password protection.

verbose ... extensive log output
    upw ... user password
    opw ... owner password
 inFile ... input pdf file
outFile ... output pdf file`

	usageChangeUserPW     = "usage: pdfcpu changeupw [-verbose] [-opw ownerpw] inFile upwOld upwNew"
	usageLongChangeUserPW = `Changeupw changes the user password.
	
verbose ... extensive log output
    opw ... owner password, required unless = ""
 inFile ... input pdf file
 upwOld ... old user password
 upwNew ... new user password`

	usageChangeOwnerPW     = "usage: pdfcpu changeopw [-verbose] [-upw userpw] inFile opwOld opwNew"
	usageLongChangeOwnerPW = `Changeopw changes the owner password.
	
verbose ... extensive log output
    upw ... user password, required unless = ""
 inFile ... input pdf file
 opwOld ... old owner password (supply user password on initial changeopw)
 opwNew ... new owner password`

	usageVersion     = "usage: pdfcpu version"
	usageLongVersion = "Version prints the pdfcpu version"
)
