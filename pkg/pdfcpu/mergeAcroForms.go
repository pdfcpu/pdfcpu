/*
Copyright 2021 The pdfcpu Authors.

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

// NOTE
// A naive first stab at merging AcroForms.
// Your mileage may vary.

func handleNeedAppearances(ctxSource *Context, dSrc, dDest Dict) error {
	o, found := dSrc.Find("NeedAppearances")
	if !found || o == nil {
		return nil
	}
	b, err := ctxSource.DereferenceBoolean(o, V10)
	if err != nil {
		return err
	}
	if b != nil && *b {
		dDest["NeedAppearances"] = Boolean(true)
	}
	return nil
}

func handleSigFields(ctxSource, ctxDest *Context, dSrc, dDest Dict) error {
	o, found := dSrc.Find("SigFields")
	if !found {
		return nil
	}
	iSrc, err := ctxSource.DereferenceInteger(o)
	if err != nil {
		return err
	}
	if iSrc == nil {
		return nil
	}
	// Merge SigFields into dDest.
	o, found = dDest.Find("SigFlags")
	if !found {
		dDest["SigFields"] = Integer(*iSrc)
		return nil
	}
	iDest, err := ctxDest.DereferenceInteger(o)
	if err != nil {
		return err
	}
	if iDest == nil {
		dDest["SigFields"] = Integer(*iSrc)
		return nil
	}
	// SignaturesExist
	if *iSrc&1 > 0 {
		*iDest |= 1
	}
	// AppendOnly
	if *iSrc&2 > 0 {
		*iDest |= 2
	}
	return nil
}

func handleCO(ctxSource, ctxDest *Context, dSrc, dDest Dict) error {
	o, found := dSrc.Find("CO")
	if !found {
		return nil
	}
	arrSrc, err := ctxSource.DereferenceArray(o)
	if err != nil {
		return err
	}
	o, found = dDest.Find("CO")
	if !found {
		dDest["CO"] = arrSrc
		return nil
	}
	arrDest, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arrDest) == 0 {
		dDest["CO"] = arrSrc
	} else {
		arrDest = append(arrDest, arrSrc...)
		dDest["CO"] = arrDest
	}
	return nil
}

func handleDR(ctxSource, ctxDest *Context, dSrc, dDest Dict) error {
	o, found := dSrc.Find("DR")
	if !found {
		return nil
	}
	dSrc, err := ctxSource.DereferenceDict(o)
	if err != nil {
		return err
	}
	if len(dSrc) == 0 {
		return nil
	}
	o, found = dDest.Find("DR")
	if !found {
		dDest["DR"] = dSrc
	} else {
		dDest, err := ctxDest.DereferenceDict(o)
		if err != nil {
			return err
		}
		if len(dDest) == 0 {
			dDest["DR"] = dSrc
		}
	}
	return nil
}

func handleDA(ctxSource *Context, dSrc, dDest Dict, arrFieldsSrc Array) error {
	// (for each with field type  /FT /Tx w/o DA, set DA to default DA)
	// TODO Walk field tree and inspect terminal fields.

	sSrc := dSrc.StringEntry("DA")
	if sSrc == nil || len(*sSrc) == 0 {
		return nil
	}
	sDest := dDest.StringEntry("DA")
	if sDest == nil {
		dDest["DA"] = StringLiteral(*sSrc)
		return nil
	}
	// Push sSrc down to all top level fields of dSource
	for _, o := range arrFieldsSrc {
		d, err := ctxSource.DereferenceDict(o)
		if err != nil {
			return err
		}
		n := d.NameEntry("FT")
		if n != nil && *n == "Tx" {
			_, found := d.Find("DA")
			if !found {
				d["DA"] = StringLiteral(*sSrc)
			}
		}
	}
	return nil
}

func handleQ(ctxSource *Context, dSrc, dDest Dict, arrFieldsSrc Array) error {
	// (for each with field type /FT /Tx w/o Q, set Q to default Q)
	// TODO Walk field tree and inspect terminal fields.

	iSrc := dSrc.IntEntry("Q")
	if iSrc == nil {
		return nil
	}
	iDest := dDest.IntEntry("Q")
	if iDest == nil {
		dDest["Q"] = Integer(*iSrc)
		return nil
	}
	// Push iSrc down to all top level fields of dSource
	for _, o := range arrFieldsSrc {
		d, err := ctxSource.DereferenceDict(o)
		if err != nil {
			return err
		}
		n := d.NameEntry("FT")
		if n != nil && *n == "Tx" {
			_, found := d.Find("Q")
			if !found {
				d["Q"] = Integer(*iSrc)
			}
		}
	}
	return nil
}

func handleFormAttributes(ctxSource, ctxDest *Context, dSrc, dDest Dict, arrFieldsSrc Array) error {

	// NeedAppearances: try: set to true only
	if err := handleNeedAppearances(ctxSource, dSrc, dDest); err != nil {
		return err
	}

	// SigFlags: set bit 1 to true only (SignaturesExist)
	//           set bit 2 to true only (AppendOnly)
	if err := handleSigFields(ctxSource, ctxDest, dSrc, dDest); err != nil {
		return err
	}

	// CO: add all indrefs
	if err := handleCO(ctxSource, ctxDest, dSrc, dDest); err != nil {
		return err
	}

	// DR: default resource dict
	if err := handleDR(ctxSource, ctxDest, dSrc, dDest); err != nil {
		return err
	}

	// DA: default appearance streams for variable text fields
	if err := handleDA(ctxSource, dSrc, dDest, arrFieldsSrc); err != nil {
		return err
	}

	// Q: left, center, right for variable text fields
	if err := handleQ(ctxSource, dSrc, dDest, arrFieldsSrc); err != nil {
		return err
	}

	// XFA: ignore
	delete(dDest, "XFA")

	return nil
}

func mergeAcroForms(ctxSource, ctxDest *Context) error {

	rootDictDest, err := ctxDest.Catalog()
	if err != nil {
		return err
	}

	rootDictSource, err := ctxSource.Catalog()
	if err != nil {
		return err
	}

	o, found := rootDictSource.Find("AcroForm")
	if !found {
		return nil
	}

	dSrc, err := ctxSource.DereferenceDict(o)
	if err != nil || len(dSrc) == 0 {
		return err
	}

	// Retrieve ctxSrc AcroForm Fields
	o, found = dSrc.Find("Fields")
	if !found {
		return nil
	}
	arrFieldsSrc, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arrFieldsSrc) == 0 {
		return nil
	}

	// We have a ctxSrc.Acroform with fields.

	o, found = rootDictDest.Find("AcroForm")
	if !found {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}

	dDest, err := ctxDest.DereferenceDict(o)
	if err != nil {
		return err
	}

	if len(dDest) == 0 {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}

	// Retrieve ctxDest AcroForm Fields
	o, found = dDest.Find("Fields")
	if !found {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}
	arrFieldsDest, err := ctxDest.DereferenceArray(o)
	if err != nil {
		return err
	}
	if len(arrFieldsDest) == 0 {
		rootDictDest["AcroForm"] = dSrc
		return nil
	}

	// Merge Dsrc into dDest.

	// Fields: add all indrefs

	// Merge all fields from ctxSrc into ctxDest
	arrFieldsDest = append(arrFieldsDest, arrFieldsSrc...)
	dDest["Fields"] = arrFieldsDest

	return handleFormAttributes(ctxSource, ctxDest, dSrc, dDest, arrFieldsSrc)
}
