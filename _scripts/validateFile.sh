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

# eg: ./validateFile.sh ~/pdf/1mb/a.pdf ~/pdf/out

if [ $# -ne 2 ]; then
    echo "usage: ./validateFile.sh inFile logDir"
    exit 1
fi

f=${1##*/}
f1=${f%.*}
out=$2

#rm -drf $out/*

cp $1 $out/$f

pdfcpu validate -verbose -mode=relaxed $out/$f &> $out/$f1.log

if [ $? -eq 1 ]; then
    echo "validation error: $out/$f"
    exit $?
else
    echo "validation success: $out/$f"
fi