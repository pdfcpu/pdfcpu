# Security Fix Plan - Issue #1350

## Scope
Fix the **Critical** and **High** severity findings (17 issues).
Medium/Low are either PDF spec constraints or lower priority — they can be a follow-up.

---

## Phase 1: Decompression bomb protection (Critical #1)

### File: `pkg/filter/flateDecode.go`
- Add a `const maxDecompressedSize = 250 * 1024 * 1024` (250MB) in the filter package
- In `passThru()` (line 108): replace `io.Copy` with `io.CopyN(&b, rin, maxDecompressedSize+1)` and error if exceeded
- In `Encode` func (line 73): same io.Copy → bounded

### File: `pkg/filter/lzwDecode.go`
- Line 87: same pattern — replace `io.Copy(&b, rc)` with bounded copy

### File: `pkg/filter/ccittDecode.go`
- Line 92: same pattern

### File: `pkg/filter/ascii85Decode.go`
- Line 78: same pattern

---

## Phase 2: Unbounded stream allocation (Critical #2)

### File: `pkg/pdfcpu/read.go`
- Line 2379: `make([]byte, streamLength)` — add a max check (e.g. 250MB) before allocation
- Return error if streamLength exceeds limit

---

## Phase 3: Timestamp signature verification (Critical #3)

### File: `pkg/pdfcpu/sign/pkcs7.go`
- Lines 394-396: Uncomment the `VerifyWithChain` call

---

## Phase 4: XRef stream Index allocation (Critical #4)

### File: `pkg/pdfcpu/model/parse.go`
- Line 1098: Add max count check (e.g. 10_000_000) before the inner loop
- Return errXrefStreamCorruptIndex if exceeded

---

## Phase 5: Path traversal (High #5, #6, #23)

### File: `pkg/api/split.go`
- Line 179: Sanitize `bm.Title` with `filepath.Base()` before using in path

### File: `pkg/api/extract.go`
- Line 140: Sanitize `f.Name` and `f.Type` with `filepath.Base()`

### File: `pkg/pdfcpu/image.go`
- Line 281: Sanitize `img.Name` and `img.FileType` with `filepath.Base()`

---

## Phase 6: Unbounded buffers (High #7, #8)

### File: `pkg/pdfcpu/read.go`
- `readStreamContentBlindly` (line 2332): Add a max total buffer size check in the growth loop
- `buffer` (line 1800): Add same max buffer size check

---

## Phase 7: Index OOB panics (High #9, #10)

### File: `pkg/pdfcpu/read.go`
- `nextStreamOffset` (line 1745): Add bounds check before `line[off]` accesses

### File: `pkg/pdfcpu/model/parseContent.go`
- `skipTJ` (line 128): Add `len(s) == 0` check before `s[0]`
- `skipBI` (line 185): Add `len(s) < 3` check before `s[2]`
- `positionToNextContentToken` (line 237): Add `len(l) == 0` check before `l[0]`

---

## Phase 8: Recursion depth limits (High #11)

### File: `pkg/pdfcpu/model/parse.go`
- Add a `maxParseDepth = 100` constant
- Add depth parameter to `ParseObjectContext`, `ParseArray`, `ParseDict`
- Return error if depth exceeded
- This is the most invasive change — need to thread depth through call chain

---

## Phase 9: Constant-time password comparison (High #12)

### File: `pkg/pdfcpu/crypto.go`
- Import `crypto/subtle`
- Line 272: Replace `bytes.Equal(ctx.E.U, u)` with `subtle.ConstantTimeCompare(ctx.E.U, u) == 1`
- Line 275: Replace `bytes.HasPrefix(ctx.E.U, u[:16])` with `subtle.ConstantTimeCompare(ctx.E.U[:16], u[:16]) == 1`
- Line 470: Replace `bytes.HasPrefix(ctx.E.O, s[:])` with `subtle.ConstantTimeCompare(ctx.E.O[:32], s[:]) == 1`
- Line 511: Replace `bytes.HasPrefix(ctx.E.U, s[:])` with `subtle.ConstantTimeCompare(ctx.E.U[:32], s[:]) == 1`

---

## Phase 10: PKCS#7 padding validation (High #13)

### File: `pkg/pdfcpu/crypto.go`
- Line 1748: After checking last byte, validate ALL padding bytes match
- Return error on invalid padding

---

## Phase 11: Additional High fixes

### RunLengthDecode OOB (Medium #21 but easy fix)
- `pkg/filter/runLengthDecode.go` line 45: Add `i >= len(src)` bounds check

### Slice bounds in parseObjectStream (#18)
- `pkg/pdfcpu/read.go` line 457: Add bounds check before slice

### Slice bounds in DecodeLength (#19)
- `pkg/pdfcpu/types/streamdict.go` line 406: Clamp maxLen to len(sd.Content)

---

## Testing
- `go build ./...` to verify compilation
- `go test ./...` to run existing test suite
