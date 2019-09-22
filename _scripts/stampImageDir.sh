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

# eg: ./stampImageDir.sh ~/pdf/big ~/pdf/out
 
if [ $# -ne 2 ]; then
    echo "usage: ./stampImageDir.sh inDir outDir"
    echo "adds image stamps with rotation angle of 0 degrees"
    exit 1
fi

out=$2

#rm -drf $out/*

#set -e

new=_wm

for pdf in $1/*.pdf
do
	#echo $pdf
	
	f=${pdf##*/}
	#echo f = $f
	
	f1=${f%.*}
	#echo f1 = $f1
	
	cp $pdf $out/$f
	
	out1=$out/$f1$new.pdf
	pdfcpu stamp add -verbose "resources/GC2018.png, rot:0" $out/$f $out1 &> $out/$f1.log
	if [ $? -eq 1 ]; then
        echo "stamp error: $pdf -> $out1"
        echo
		continue
    else
        echo "stamp success: $pdf -> $out1"
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
