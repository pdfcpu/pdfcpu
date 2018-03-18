#!/bin/sh

# eg: ./trimDir.sh ~/pdf/big ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./trimDir.sh inDir outDir"
    echo "generate PDFs with the first 5 pages"
    exit 1
fi

out=$2

#rm -drf $out/*

#set -e

new=_trim

for pdf in $1/*.pdf
do
	#echo $pdf
	
	f=${pdf##*/}
	#echo f = $f
	
	f1=${f%.*}
	#echo f1 = $f1
	
	cp $pdf $out/$f
	
	out1=$out/$f1$new.pdf
	pdfcpu trim -verbose -pages=-5 $out/$f $out1 &> $out/$f1.log
	if [ $? -eq 1 ]; then
        echo "trim error: $pdf -> $out1"
        echo
		continue
    else
        echo "trim success: $pdf -> $out1"
		pdfcpu validate -verbose -mode=relaxed $out1 >> $out/$f1.log 2>&1
       	if [ $? -eq 1 ]; then
        	echo "validation error: $out"
            exit $?
        else
            echo "validation success: $out"
        fi
    fi
	
	echo
	
done
