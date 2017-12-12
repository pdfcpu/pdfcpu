/*

Package pdfcpu is a simple PDF processing library written in Go supporting encryption.
It provides both an API and a command line tool.
Supported are all versions up to PDF 1.7 (ISO-32000).

The available commands are:

	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)
	optimize	optimize PDF by getting rid of redundant page resources
	split		split multi-page PDF into several single-page PDFs
	merge		concatenate 2 or more PDFs
	extract		extract images, fonts, content or pages
	trim		create trimmed version
	attachments	list, add, remove, extract embedded file attachments
	encrypt		set password
	decrypt		remove password
	changeupw	change user password
	changeopw	change owner password
	version		print pdfcpu version

*/
package pdfcpu
