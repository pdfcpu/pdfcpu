#!/bin/sh

#: ./encryptFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./encryptFile.sh inFile outDir"
    exit 1
fi

new=_new

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

cp $1 $out/$f

out1=$out/$f1$new.pdf
pdfcpu encrypt -verbose -upw upw -opw opw $out/$f $out1 &> $out/$f1.log
if [ $? -eq 1 ]; then
    echo "encryption error: $1 -> $out1"
    exit $?
else
    echo "encryption success: $1 -> $out1"
fi
	
pdfcpu validate -verbose -mode=relaxed -upw upw -opw opw $out1 &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "validation error: $out1"
    exit $?
else
    echo "validation success: $out1"
fi

pdfcpu changeupw -opw opw -verbose $out1 upw upwNew &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "changeupw error: $1 -> $out1"
    exit $?
else
    echo "changeupw success: $1 -> $out1"
fi

pdfcpu validate -verbose -mode=relaxed -upw upwNew -opw opw $out1 &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "validation error: $out1"
    exit $?
else
    echo "validation success: $out1"
fi

pdfcpu changeopw -upw upwNew -verbose $out1 opw opwNew &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "changeopw error: $1 -> $out1"
    exit $?
else
    echo "changeopw success: $1 -> $out1"
fi

pdfcpu validate -verbose -mode=relaxed -upw upwNew -opw opwNew $out1 &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "validation error: $out1"
    exit $?
else
    echo "validation success: $out1"
fi

pdfcpu decrypt -verbose -upw upwNew -opw opwNew $out1 $out1 &> $out/$f1.log
if [ $? -eq 1 ]; then
    echo "decryption error: $out1 -> $out1"
    exit $?
else
    echo "decryption success: $out1 -> $out1"
fi
	
pdfcpu validate -verbose -mode=relaxed $out1 &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "validation error: $out1"
    exit $?
else
    echo "validation success: $out1"
fi

	
