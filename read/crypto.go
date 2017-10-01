package read

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rc4"
	"io"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var pad = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41, 0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80, 0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

type encrypt struct {
	o, u       []byte
	l, p, r, v int
	emd        bool
}

func validateUserPassword(userpw string, e *encrypt, id []byte) (ok bool, key []byte, err error) {

	// fmt.Printf("id: %X\n", id)
	// fmt.Printf(" o: %X\n", e.o)
	// fmt.Printf(" u: %X\n", e.u)
	// fmt.Printf(" l:%d p:%d r:%d  v:%d emd:%t\n", e.l, e.p, e.r, e.v, e.emd)

	// Alg.4/5 p63

	// 4a/5a create enryption key using Alg.2 p61

	// 2a
	pw := []byte(userpw)
	if len(pw) >= 32 {
		pw = pw[:32]
	} else {
		pw = append(pw, pad[:32-len(pw)]...)
	}

	// Create an enryption key Algor 2
	// 2b
	h := md5.New()
	h.Write(pw)

	// 2c
	h.Write(e.o)

	// 2d
	var q = uint32(e.p)
	h.Write([]byte{byte(q), byte(q >> 8), byte(q >> 16), byte(q >> 24)})

	// 2e
	h.Write(id)

	// 2f
	if e.r == 4 && !e.emd {
		h.Write([]byte{0xff, 0xff, 0xff, 0xff})
	}

	// 2g
	key = h.Sum(nil)
	//fmt.Printf("start key: % x\n\n", key)

	// 2h
	if e.r >= 3 {
		for i := 0; i < 50; i++ {
			h.Reset()
			h.Write(key[:e.l/8])
			key = h.Sum(key[:0])
			//fmt.Printf("%02d: % x\n", i, key)
		}
	}

	// 2i
	if e.r >= 3 {
		key = key[:e.l/8]
	} else {
		key = key[:5]
	}
	//fmt.Printf("\nencryption key after 50 md5 iterations:\n%X\n\n", key)

	c, err := rc4.NewCipher(key)
	if err != nil {
		return false, nil, err
	}

	var u []byte

	if e.r == 2 {
		// 4b
		u = make([]byte, 32)
		copy(u, pad)
		c.XORKeyStream(u, u)
	} else {

		// 5b
		h.Reset()
		h.Write(pad)

		// 5c
		h.Write(id)
		u = h.Sum(nil)

		// 5ds
		c.XORKeyStream(u, u)

		// 5e
		//fmt.Printf("u: %X\n", u)

		for i := 1; i <= 19; i++ {
			keynew := make([]byte, len(key))
			copy(keynew, key)

			for j := range keynew {
				keynew[j] ^= byte(i)
			}

			c, err := rc4.NewCipher(keynew)
			if err != nil {
				return false, nil, err
			}
			c.XORKeyStream(u, u)
			//fmt.Printf("%02d: %X\n", i, u)
		}
	}

	//fmt.Printf("\ncalculated u: %X\n", u)
	//fmt.Printf("u:            %X\n", e.u)

	return bytes.HasPrefix(e.u, u), key, nil
}

func supportedCFEntry(d *types.PDFDict) (bool, bool) {

	cfm := d.NameEntry("CFM")
	if cfm != nil && *cfm != "V2" && *cfm != "AESV2" {
		logErrorReader.Println("supportedCFEntry: invalid entry \"CFM\"")
		return false, false
	}

	ae := d.NameEntry("AuthEvent")
	if ae != nil && *ae != "DocOpen" {
		logErrorReader.Println("supportedCFEntry: invalid entry \"AuthEvent\"")
		return false, false
	}

	l := d.IntEntry("Length")
	if l != nil && (*l < 8 || *l > 128 || *l%8 > 1) {
		logErrorReader.Println("supportedCFEntry: invalid entry \"Length\"")
		return false, false
	}

	return cfm != nil && *cfm == "AESV2", true
}

func supportedEncryption(ctx *types.PDFContext, dict *types.PDFDict) (*encrypt, error) {

	var aes, ok bool
	var err error

	// Filter
	filter := dict.NameEntry("Filter")
	if filter == nil || *filter != "Standard" {
		logErrorReader.Println("supportedEncryption: Filter must be \"Standard\"")
		return nil, nil
	}

	// SubFilter
	if dict.NameEntry("SubFilter") != nil {
		logErrorReader.Println("supportedEncryption: \"SubFilter\" not supported")
		return nil, nil
	}

	// V
	v := dict.IntEntry("V")
	if v == nil || (*v != 1 && *v != 2 && *v != 4) {
		logErrorReader.Println("supportedEncryption: \"V\" must be one of 1,2,4")
		return nil, nil
	}

	// Length
	length := 40
	l := dict.IntEntry("Length")
	if l != nil {
		if *l < 40 || *l > 128 || *l%8 > 0 {
			logErrorReader.Printf("supportedEncryption: \"Length\" %d not supported\n", *l)
			return nil, nil
		}
		length = *l
	}

	if *v == 4 {

		// CF
		cfDict := dict.PDFDictEntry("CF")
		if cfDict == nil {
			logErrorReader.Println("supportedEncryption: required entry \"CF\" missing.")
			return nil, nil
		}

		// StmF
		stmf := dict.NameEntry("StmF")
		if stmf != nil && *stmf != "Identity" {
			d := cfDict.PDFDictEntry(*stmf)
			if d == nil {
				logErrorReader.Printf("supportedEncryption: entry \"%s\" missing in \"CF\"", *stmf)
				return nil, nil
			}
			aes, ok = supportedCFEntry(d)
			if !ok {
				return nil, errors.Errorf("supportedEncryption: unsupported \"%s\" entry in \"CF\"", *stmf)
			}
			ctx.AES4Streams = aes
		}

		// StrF
		strf := dict.NameEntry("StrF")
		if strf != nil && *strf != "Identity" {
			d := cfDict.PDFDictEntry(*strf)
			if d == nil {
				logErrorReader.Printf("supportedEncryption: entry \"%s\" missing in \"CF\"", *strf)
				return nil, nil
			}
			aes, ok = supportedCFEntry(d)
			if !ok {
				return nil, errors.Errorf("supportedEncryption: unsupported \"%s\" entry in \"CF\"", *strf)
			}
			ctx.AES4Strings = aes
		}

		// EFF
		eff := dict.NameEntry("EFF")
		if eff != nil && *strf != "Identity" {
			d := cfDict.PDFDictEntry(*eff)
			if d == nil {
				logErrorReader.Printf("supportedEncryption: entry \"%s\" missing in \"CF\"", *eff)
				return nil, nil
			}
			aes, ok = supportedCFEntry(d)
			if !ok {
				return nil, errors.Errorf("supportedEncryption: unsupported \"%s\" entry in \"CF\"", *eff)
			}
			ctx.AES4EmbeddedStreams = aes
		}

	}

	// R
	r := dict.IntEntry("R")
	if r == nil || (*r != 2 && *r != 3 && *r != 4) {
		logErrorReader.Println("supportedEncryption: \"R\" must be 2,3,4")
		return nil, nil
	}

	// O
	o, err := dict.StringEntryBytes("O")
	if err != nil {
		return nil, err
	}
	if o == nil || len(o) != 32 {
		logErrorReader.Println("supportedEncryption: required entry \"O\" missing or invalid")
		return nil, nil
	}
	//logDebugReader.Printf("O: %X %s len:%d\n", o, o, len(o))

	// U
	u, err := dict.StringEntryBytes("U")
	if err != nil {
		return nil, err
	}
	if u == nil || len(u) != 32 {
		logErrorReader.Printf("supportedEncryption: required entry \"U\" missing or invalid %d", len(u))
		return nil, nil
	}
	//logDebugReader.Printf("U: %X %s len:%d\n", u, u, len(u))

	// P
	p := dict.IntEntry("P")
	if p == nil {
		logErrorReader.Println("supportedEncryption: required entry \"P\" missing")
		return nil, nil
	}

	// EncryptMetadata
	encMeta := true
	emd := dict.BooleanEntry("EncryptMetadata")
	if emd != nil {
		encMeta = *emd
	}

	return &encrypt{o, u, length, *p, *r, *v, encMeta}, nil
}

func checkForEncryption(ctx *types.PDFContext) error {

	indRef := ctx.Encrypt
	if indRef == nil {
		return nil
	}

	logDebugReader.Printf("Encryption: %v\n", indRef)

	obj, err := dereferencedObject(ctx, indRef.ObjectNumber.Value())
	if err != nil {
		return err
	}

	encryptDict, ok := obj.(types.PDFDict)

	if !ok {
		return errors.New("corrupt encrypt dict")
	}

	logDebugReader.Printf("%s\n", encryptDict)

	enc, err := supportedEncryption(ctx, &encryptDict)
	if err != nil {
		return err
	}
	if enc == nil {
		return errors.New("This encryption is not supported")
	}

	if ctx.ID == nil {
		return errors.New("missing ID entry")
	}
	hex, ok := ((*ctx.ID)[0]).(types.PDFHexLiteral)
	if !ok {
		return errors.New("corrupt encrypt dict")
	}
	id, err := hex.Bytes()
	if err != nil {
		return err
	}

	ok, key, err := validateUserPassword("", enc, id)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("Authentication error")
	}

	ctx.EncKey = key

	return nil
}

func decryptKey(objNumber, generation int, key []byte, aes bool) []byte {

	logDebugReader.Printf("decryptKey: obj:%d gen:%d key:%x aes:%t\n", objNumber, generation, key, aes)

	m := md5.New()

	nr := uint32(objNumber)
	b1 := []byte{byte(nr), byte(nr >> 8), byte(nr >> 16)}
	b := append(key, b1...)

	gen := uint16(generation)
	b2 := []byte{byte(gen), byte(gen >> 8)}
	b = append(b, b2...)

	logDebugReader.Printf("b: %X\n", b)

	m.Write(b)

	if aes {
		m.Write([]byte("sAlT"))
	}

	dk := m.Sum(nil)

	l := len(key) + 5
	if l < 16 {
		dk = dk[:l]
	}

	logDebugReader.Printf("decryptKey returning: %X\n", dk)

	return dk
}

func decryptString(s string, objNr, genNr int, key []byte, needAES bool) (*string, error) {

	logDebugReader.Printf("decryptString begin s:<%s> %d %d key:%X aes:%t\n", s, objNr, genNr, key, needAES)

	k := decryptKey(objNr, genNr, key, needAES)

	b, err := types.Unescape(s)
	if err != nil {
		return nil, err
	}

	logDebugReader.Printf("decryptString unescaped: %X %d\n", b, len(b))

	if needAES {

		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, err
		}

		if len(b) < aes.BlockSize {
			return nil, errors.New("decryptStream: aes ciphertext too short")
		}

		iv := make([]byte, 16)
		copy(iv, b[:16])

		data := b[16:]

		if len(data)%aes.BlockSize != 0 {
			return nil, errors.New("decryptStream: aes ciphertext not a multiple of block size")
		}

		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(data, data)
		s1 := string(data)
		logDebugReader.Printf("decryptString end, returning: <%s>\n", s1)
		//panic("string game over!")

		return &s1, nil
	}

	c, err := rc4.NewCipher(decryptKey(objNr, genNr, key, needAES))
	if err != nil {
		return nil, err
	}

	c.XORKeyStream(b, b)
	s1 := string(b)
	logDebugReader.Printf("decryptString end, returning: <%s>\n", s1)
	//panic("game over!")

	return &s1, nil
}

func decryptDeepObject(objIn interface{}, objNr, genNr int, key []byte, aes bool) (*types.PDFStringLiteral, error) {

	//logDebugReader.Printf("decryptDeepObject: <%v> %T\n", objIn, objIn)

	_, ok := objIn.(types.PDFIndirectRef)
	if ok {
		return nil, nil
	}

	switch obj := objIn.(type) {

	case types.PDFDict:
		for k, v := range obj.Dict {
			s, err := decryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj.Dict[k] = *s
			}
		}

	case types.PDFArray:
		for i, v := range obj {
			s, err := decryptDeepObject(v, objNr, genNr, key, aes)
			if err != nil {
				return nil, err
			}
			if s != nil {
				obj[i] = *s
			}
		}

	case types.PDFStringLiteral:
		s, err := decryptString(obj.Value(), objNr, genNr, key, aes)
		if err != nil {
			return nil, err
		}

		sl := types.PDFStringLiteral(*s)

		return &sl, nil

	default:
		//logDebugReader.Printf("decryptDeepObject: obj=%T\n", obj)

	}

	return nil, nil
}

func decryptDictStrings(dict *types.PDFDict, objNr, genNr int, key []byte, aes bool) error {

	_, err := decryptDeepObject(*dict, objNr, genNr, key, aes)
	return err
}

func decryptStream(buf []byte, objNr, genNr int, key []byte, needAES bool) ([]byte, error) {

	logDebugReader.Printf("decryptStream begin obj:%d gen:%d key:%X aes:%t\n", objNr, genNr, key, needAES)

	k := decryptKey(objNr, genNr, key, needAES)

	if needAES {

		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, err
		}

		if len(buf) < aes.BlockSize {
			return nil, errors.New("decryptStream: aes ciphertext too short")
		}

		iv := make([]byte, 16)
		copy(iv, buf[:16])
		//stream := cipher.NewOFB(block, iv)

		data := buf[16:]
		if len(data)%aes.BlockSize != 0 {
			return nil, errors.New("decryptStream: aes ciphertext not a multiple of block size")
		}

		//r := bytes.NewReader(data)
		//var b bytes.Buffer

		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(data, data)
		//panic("stream game over!")

		//rd := &cipher.StreamReader{S: stream, R: r}

		// Copy the input file to the output file, decrypting as we go.
		//if _, err := io.Copy(&b, rd); err != nil {
		//	return nil, err
		//}

		return data, nil
		//return b.Bytes(), nil
	}

	c, err := rc4.NewCipher(k)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(buf)
	var b bytes.Buffer

	rd := &cipher.StreamReader{S: c, R: r}

	if _, err = io.Copy(&b, rd); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func decryptStreamFix(buf []byte, k []byte, needAES bool) ([]byte, error) {

	logDebugReader.Printf("decryptStreamFix begin key:%X aes:%t\n", k, needAES)

	if needAES {

		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, err
		}

		if len(buf) < aes.BlockSize {
			return nil, errors.New("decryptStream: aes ciphertext too short")
		}

		iv := make([]byte, 16)
		copy(iv, buf[:16])
		//stream := cipher.NewOFB(block, iv)

		data := buf[16:]
		if len(data)%aes.BlockSize != 0 {
			return nil, errors.New("decryptStream: aes ciphertext not a multiple of block size")
		}

		//r := bytes.NewReader(data)
		//var b bytes.Buffer

		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(data, data)
		//panic("stream game over!")

		//rd := &cipher.StreamReader{S: stream, R: r}

		// Copy the input file to the output file, decrypting as we go.
		//if _, err := io.Copy(&b, rd); err != nil {
		//	return nil, err
		//}

		return data, nil
		//return b.Bytes(), nil
	}

	c, err := rc4.NewCipher(k)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(buf)
	var b bytes.Buffer

	rd := &cipher.StreamReader{S: c, R: r}

	if _, err = io.Copy(&b, rd); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
