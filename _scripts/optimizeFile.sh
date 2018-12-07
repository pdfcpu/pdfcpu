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

# eg: ./optimizeFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./optimizeFile.sh inFile outDir"
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
pdfcpu optimize -verbose $out/$f $out1 &> $out/$f1.log
if [ $? -eq 1 ]; then
    echo "optimization error: $1 -> $out1"
    exit $?
else
    echo "optimization success: $1 -> $out1"
fi
	
out2=$out/$f1$new$new.pdf
pdfcpu optimize -verbose $out1 $out2 &> $out/$f1$new.log
if [ $? -eq 1 ]; then
    echo "optimization error: $out1 -> $out2"
    exit $?
else
    echo "optimization success: $out1 -> $out2"
fi

	
