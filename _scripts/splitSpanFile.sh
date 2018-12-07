#!/bin/sh

# Copyright 2018 The pdfcpu Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# eg: ./splitSpanFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./splitSpanFile.sh inFile outDir"
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

#set -e

mkdir $out/$f1
cp $1 $out/$f1 

# Split this file up by generating a new PDF for every 2 pages. 
span=2

pdfcpu split -verbose $out/$f1/$f $out/$f1 $span &> $out/$f1/$f1.log
if [ $? -eq 1 ]; then
    echo "split error: $1 -> $out"
    exit $?
else
    echo "split success: $1 -> $out"
    for pdf in $out/$f1/*_*.pdf
    do
        echo "validating: $pdf"
        pdfcpu validate -verbose -mode=relaxed $pdf >> $out/$f1/$f1.log 2>&1
        if [ $? -eq 1 ]; then
            echo "validation error: $pdf"
            exit $?
        #else
            #echo "validation success: $pdf"
        fi
    done
fi
	

	
