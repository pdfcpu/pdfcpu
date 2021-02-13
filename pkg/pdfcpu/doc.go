/*

Package pdfcpu is a PDF processing library written in Go supporting encryption.
It provides an API and a command line interface. Supported are all versions up to PDF 1.7 (ISO-32000).

The commands are:

	attachments	list, add, remove, extract embedded file attachments
	booklet     arrange pages onto larger sheets of paper to make a booklet or zine
	boxes       list, add, remove page boundaries for selected pages
	changeopw	change owner password
	changeupw	change user password
	collect		create custom sequence of selected pages
	crop        set cropbox for selected pages
	decrypt		remove password protection
	encrypt		set password protection
	extract		extract images, fonts, content, pages or metadata
	fonts		install, list supported fonts, create cheat sheets
	grid		rearrange pages or images for enhanced browsing experience
	import		import/convert images to PDF
	info		print file info
	keywords	list, add, remove keywords
	merge		concatenate PDFs
	nup			rearrange pages or images for reduced number of pages
	optimize	optimize PDF by getting rid of redundant page resources
	pages		insert, remove selected pages
	paper		print list of supported paper sizes
	permissions	list, set user access permissions
	portfolio	list, add, remove, extract portfolio entries with optional description
	properties	list, add, remove document properties
	rotate		rotate pages
	split		split up a PDF by span or bookmark
	stamp		add, remove, update Unicode text, image or PDF stamps for selected pages
	trim		create trimmed version of selected pages
	validate	validate PDF against PDF 32000-1:2008 (PDF 1.7)
	version		print version
	watermark	add, remove, update Unicode text, image or PDF watermarks for selected pages

*/
package pdfcpu
