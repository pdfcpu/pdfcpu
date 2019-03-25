/*

Package pdfcpu is a simple PDF processing library written in Go supporting encryption.
It provides an API and a command line interface. Supported are all versions up to PDF 1.7 (ISO-32000).

The commands are:

	attachments	list, add, remove, extract embedded file attachments
	changeopw	change owner password
	changeupw	change user password
	decrypt		remove password protection
	encrypt		set password protection
	extract		extract images, fonts, content, pages or metadata
	grid		rearrange pages orimages for enhanced browsing experience
	import		import/convert images
	merge		concatenate 2 or more PDFs
	nup			rearrange pages or images for reduced number of pages
	optimize	optimize PDF by getting rid of redundant page resources
	pages		insert, remove selected pages
	paper		print list of supported paper sizes
	permissions	list, add user access permissions
	rotate		rotate pages
	split		split multi-page PDF into several PDFs according to split span
	stamp		add text, image or PDF stamp to selected pages
	trim		create trimmed version with selected pages
	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)
	version		print version
	watermark	add text, image or PDF watermark to selected pages

*/
package pdfcpu
