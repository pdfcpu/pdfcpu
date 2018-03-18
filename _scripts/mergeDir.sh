#!/bin/sh

# eg: ./mergeDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./mergeDir.sh inDir outDir"
    exit 1
fi

out=$2

#rm -drf $out/*

#set -e

for pdf in $1/*.pdf
do
	f=${pdf##*/}
	cp $pdf $out/$f
done

pdfcpu merge -verbose $out/merged.pdf $out/*.pdf &> $out/merged.log
if [ $? -eq 1 ]; then
	echo "merge error: $1/*.pdf -> $out/merged.pdf"
    echo
	continue
else
    echo "merge success: $1/*.pdf -> $out/merged.pdf"
fi