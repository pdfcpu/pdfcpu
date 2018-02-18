/*

Package pdfcpu is a simple PDF processing library written in Go supporting encryption.
It provides an API and a command line interface. Supported are all versions up to PDF 1.7 (ISO-32000).

The available commands are:

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

*/
package pdfcpu
