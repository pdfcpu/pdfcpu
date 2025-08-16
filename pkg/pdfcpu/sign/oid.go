/*
Copyright 2025 The pdfcpu Authors.

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

import "encoding/asn1"

var (
	oidETSIQCPublicWithSSCD    = asn1.ObjectIdentifier{0, 4, 0, 1456, 1, 1}                // Qualified Certificate ETSI
	oidSigPolicy               = asn1.ObjectIdentifier{0, 4, 0, 2023, 1, 1}                // ETSI Qualified Signature Policy (for EU Qualified Electronic Signatures)
	oidQualSealPolicy          = asn1.ObjectIdentifier{0, 4, 0, 2023, 1, 2}                // ETSI Qualified Seal Policy (for legal entity seals)
	oidAdvSigPolicy            = asn1.ObjectIdentifier{0, 4, 0, 2023, 1, 3}                // ETSI Advanced Signature Policy (for advanced e-signatures)
	oidAdvSigLTVPolicy         = asn1.ObjectIdentifier{0, 4, 0, 2023, 1, 4}                // Advanced Signature with long term validation support
	oidQESLTVPolicy            = asn1.ObjectIdentifier{0, 4, 0, 2023, 1, 5}                // QES with LTV (qualified + long-term archive)
	oidQCESign                 = asn1.ObjectIdentifier{0, 4, 0, 194112, 1, 2}              // Qualified Certificate for Electronic Signatures
	oidQCESeal                 = asn1.ObjectIdentifier{0, 4, 0, 194112, 1, 3}              // Qualified Certificate for Electronic Seals
	oidQWebAuthCert            = asn1.ObjectIdentifier{0, 4, 0, 194112, 1, 4}              // Web Authentication Certificate
	oidRSAESOAEP               = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 10}        // RSAES-OAEP
	oidData                    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}         // PAdES-E-BES content-type, signed
	oidMessageDigest           = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}         // PAdES-E-BES, signed
	oidSigningTime             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}         // PKSC#7, signed
	oidTSTInfo                 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}  // Time Stamp Token Information, ETSI.RFC3161
	oidSigningCertificate      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 12} // PAdES-E-BES, signed
	oidTimestampToken          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 14} // PAdES-T, unsigned
	oidSigPolicyID             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 15} // PAdES-EPES, signed
	oidCommitmentType          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 16} // PAdES-EPES, signed
	oidContentTimestamp        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 20} // PAdES-T, signed
	oidCompleteCertificateRefs = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 21} // CAdES-C, unsigned
	oidCompleteRevocationRefs  = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 22} // CAdES-C, unsigned
	oidCertificateValues       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 23} // CAdES-X, unsigned
	oidRevocationValues        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 24} // CAdES-X, unsigned
	oidArchiveTimestamp        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 27} // CAdES-A, unsigned
	oidSigningCertificateV2    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 2, 47} // PAdES-E-BES, signed
	oidProofOfOrigin           = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 1}  // Signer claims authorship
	oidProofOfReceipt          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 2}  // Signer acknowledges receipt
	oidProofOfDelivery         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 3}  // Signer confirms delivery
	oidProofOfSender           = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 4}  // Signer confirms they sent the data
	oidProofOfApproval         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 5}  // Signer approves content
	oidProofOfCreation         = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 6, 6}  // Signer created the content
	oidRevocationInfoArchival  = asn1.ObjectIdentifier{1, 2, 840, 113583, 1, 1, 8}         // Embedded revocation data, signed
	oidOCSPNoCheck             = asn1.ObjectIdentifier{1, 3, 6, 1, 5, 5, 7, 48, 1, 5}      // OSCP responder cert extension
)
