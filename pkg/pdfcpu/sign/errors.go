/*
Copyright 2026 The pdfcpu Authors.

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

package sign

import (
	"errors"
	"fmt"
)

var (
	errCertificateParse = errors.New("certificate parse error")

	errMalformedByteRange = errors.New("malformed signature ByteRange")

	errUnsupportedPublicKey = errors.New("unsupported public key algorithm")

	errMalformedPublicKey = errors.New("malformed public key")

	errESSCertificateMismatch = errors.New("ESS: signing certificate mismatch")

	errUnsupportedESSCertificateProfile = errors.New("ESS: unsupported signing certificate profile")

	errMalformedAdobePKCS7Profile = errors.New("Adobe PKCS#7: malformed profile")

	errMalformedCAdESBaselineBProfile = errors.New("CAdES baseline B: malformed profile")

	errUnsupportedCAdESBaselineBProfile = errors.New("CAdES baseline B: unsupported profile")

	errCAdESCertificateBindingMismatch = errors.New("CAdES baseline B: certificate binding mismatch")
)

type certificateParseError struct {
	cause error
}

func (e *certificateParseError) Error() string {
	return fmt.Sprintf("%s: %v", errCertificateParse, e.cause)
}

func (e *certificateParseError) Is(target error) bool {
	return target == errCertificateParse
}

func (e *certificateParseError) Unwrap() error {
	return e.cause
}

type malformedByteRangeError struct {
	cause error
}

func (e *malformedByteRangeError) Error() string {
	return e.cause.Error()
}

func (e *malformedByteRangeError) Is(target error) bool {
	return target == errMalformedByteRange
}

func (e *malformedByteRangeError) Unwrap() error {
	return e.cause
}

type byteRangeReadError struct {
	cause error
}

func (e *byteRangeReadError) Error() string {
	return e.cause.Error()
}

func (e *byteRangeReadError) Unwrap() error {
	return e.cause
}

func malformedByteRange(cause error) error {
	return &malformedByteRangeError{cause: cause}
}

func fatalByteRangeRead(cause error) error {
	return &byteRangeReadError{cause: cause}
}
