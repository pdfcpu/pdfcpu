/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pdfcpu

import "testing"

func Test_decodePixelValue(t *testing.T) {
	type args struct {
		v   uint8
		bpc int
		r   colValRange
	}
	tests := []struct {
		name string
		args args
		want uint8
	}{
		{
			name: "1 bit black [0..1]",
			args: args{v: 0, bpc: 1, r: colValRange{0, 1}},
			want: 0,
		},
		{
			name: "1 bit white [0..1]",
			args: args{v: 1, bpc: 1, r: colValRange{0, 1}},
			want: 1,
		},
		{
			name: "1 bit black [0..255]",
			args: args{v: 0, bpc: 1, r: colValRange{0, 255}},
			want: 0,
		},
		{
			name: "1 bit white [0..255]",
			args: args{v: 1, bpc: 1, r: colValRange{0, 255}},
			want: 255,
		},
		// TODO more combinations
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodePixelValue(tt.args.v, tt.args.bpc, tt.args.r); got != tt.want {
				t.Errorf("decodePixelValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
