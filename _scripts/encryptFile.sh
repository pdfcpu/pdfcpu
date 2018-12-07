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

# eg: ./encryptFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

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

	
