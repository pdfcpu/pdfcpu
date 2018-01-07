package types

import (
	"github.com/pkg/errors"
)

// PDFNameTree represents a PDF name tree.
// See 7.9.6
type PDFNameTree struct {
	PDFIndirectRef
}

// NewNameTree creates a new nameTree.
func NewNameTree(root PDFIndirectRef) *PDFNameTree {
	return &PDFNameTree{root}
}

func (nameTree PDFNameTree) rootObjNr() int {
	return nameTree.ObjectNumber.Value()
}

func (nameTree PDFNameTree) namesArray(xRefTable *XRefTable) (arr *PDFArray, err error) {

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return
	}

	obj, ok := dict.Find("Names")
	if !ok {
		return nil, errors.Errorf("namesArray: missing \"Names\" entry in <%v>\n", obj)
	}

	arr, err = xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if len(*arr)%2 > 0 {
		err = errors.Errorf("limitsArray: corrupt \"Names\" entry in %v\n", *arr)
	}

	return
}

func (nameTree PDFNameTree) limits(arr *PDFArray) (min, max string, err error) {

	if len(*arr) != 2 {
		err = errors.Errorf("limits: corrupt \"Limits\" entry in %v\n", *arr)
		return
	}

	sl, ok := (*arr)[0].(PDFStringLiteral)
	if !ok {
		err = errors.Errorf("limits: corrupt min key <%v>\n", (*arr)[0])
		return
	}
	min, err = StringLiteralToString(sl.Value())
	if err != nil {
		return
	}

	sl, ok = (*arr)[1].(PDFStringLiteral)
	if !ok {
		err = errors.Errorf("limits: corrupt max key <%v>\n", (*arr)[1])
		return
	}
	max, err = StringLiteralToString(sl.Value())
	if err != nil {
		return
	}

	return
}

func (nameTree PDFNameTree) limitsArray(xRefTable *XRefTable) (arr *PDFArray, min, max string, err error) {

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return
	}

	obj, ok := dict.Find("Limits")
	if !ok {
		err = errors.Errorf("limitsArray: missing \"Limits\" entry in <%v>\n", dict)
		return
	}

	arr, err = xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	min, max, err = nameTree.limits(arr)

	return
}

// LeafNode retrieves the leaf node dict for a given key which may be a new one.
// last is true for the right most kid assuming strict left to right tree traversal.
func (nameTree *PDFNameTree) LeafNode(xRefTable *XRefTable, last bool, key string) (leaf *PDFNameTree, err error) {

	logInfoTypes.Printf("LeafNode: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	// A node has "Kids" or "Names" entry.

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		var arr *PDFArray

		// Check Limits array if present (intermediate nodes).
		if o, found := dict.Find("Limits"); found {

			arr, err = xRefTable.DereferenceArray(o)
			if err != nil {
				return
			}
			if len(*arr) != 2 {
				return nil, errors.Errorf("LeafNode: corrupt \"Limits\" entry: %v\n", *arr)
			}

			maxKey := (*arr)[1].(PDFStringLiteral).Value()
			// If key > maxKey skip this subtree unless it is the last one.
			if !last && key > maxKey {
				return nil, nil
			}

		}

		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return
		}
		if arr == nil {
			return nil, errors.New("LeafNode: missing \"Kids\" array")
		}

		kidCount := len(*arr)

		for i, obj := range *arr {

			logInfoTypes.Printf("LeafNode: processing kid: %v\n", obj)

			kid, ok := obj.(PDFIndirectRef)
			if !ok {
				return nil, errors.New("LeafNode: corrupt kid")
			}

			leaf, err = NewNameTree(kid).LeafNode(xRefTable, i == kidCount-1, key)
			if err != nil {
				return
			}
			if leaf != nil {
				return
			}

		}

		logInfoTypes.Println("LeafNode end")

		return
	}

	// Leaf node
	leaf = nameTree

	logInfoTypes.Println("LeafNode end")

	return
}

func (nameTree *PDFNameTree) key(xRefTable *XRefTable, o interface{}) (k string, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil {
		return
	}

	sl, ok := o.(PDFStringLiteral)
	if !ok {
		err = errors.Errorf("corrupt key <%v>\n", o)
		return
	}

	return StringLiteralToString(sl.Value())
}

// LeafNodeValue retrieves the indRef value for a given key of a leaf node.
// Will return nil if key not found.
func (nameTree *PDFNameTree) LeafNodeValue(xRefTable *XRefTable, key string) (indRef *PDFIndirectRef, err error) {

	logInfoTypes.Printf("LeafNodeValue: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	var arr *PDFArray
	arr, err = nameTree.namesArray(xRefTable)
	if err != nil {
		return
	}

	logInfoTypes.Printf("arr = %v\n", arr)

	var getValue bool

	for i, obj := range *arr {

		if i%2 == 0 {

			var k string
			k, err = nameTree.key(xRefTable, obj)
			if err != nil {
				return
			}

			//logErrorTypes.Printf("<%s> <%s> %0X %0x\n", s, key, s, key)

			if key == k {
				getValue = true
			}

			continue
		}

		if getValue {
			iRef, ok := obj.(PDFIndirectRef)
			if !ok {
				return nil, errors.Errorf("LeafNodeValue: corrupt value <%v>\n", obj)
			}
			indRef = &iRef
			logInfoTypes.Println("LeafNodeValue end")
			return
		}

	}

	logInfoTypes.Println("LeafNodeValue end: not found")
	return
}

// LeafNodeSetValue adds or updates a key value pair.
func (nameTree *PDFNameTree) LeafNodeSetValue(xRefTable *XRefTable, key string, val PDFIndirectRef) (err error) {

	logInfoTypes.Printf("LeafNodeSetValue: obj#%d key=%s val=%v\n", nameTree.rootObjNr(), key, val)

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return
	}

	obj, ok := dict.Find("Names")
	if !ok {
		return errors.Errorf("LeafNodeSetValue: missing \"Names\" entry in <%v>\n", obj)
	}

	var arr *PDFArray
	arr, err = xRefTable.DereferenceArray(obj)
	if err != nil {
		return
	}

	if len(*arr)%2 > 0 {
		return errors.Errorf("LeafNodeSetValue: corrupt \"Names\" entry in %v\n", *arr)
	}

	a := *arr

	newArr := PDFArray{}

	var found bool
	for i := 0; i < len(a)/2; i++ {

		keyObj := a[i*2]
		valObj := a[i*2+1]

		var k string
		k, err = nameTree.key(xRefTable, keyObj)
		if err != nil {
			return
		}

		if !found {

			if key < k {

				newArr = append(newArr, PDFStringLiteral(key))
				newArr = append(newArr, val)
				newArr = append(newArr, keyObj)
				newArr = append(newArr, valObj)

				found = true
				continue
			}

			if key == k {

				// Free up possible obj for original key.
				err = xRefTable.DeleteObjectGraph(keyObj)
				if err != nil {
					return
				}

				// Free up all objs referred by val.
				err = xRefTable.DeleteObjectGraph(valObj)
				if err != nil {
					return
				}

				newArr = append(newArr, PDFStringLiteral(key))
				newArr = append(newArr, val)

				found = true
				continue
			}

		}

		newArr = append(newArr, keyObj)
		newArr = append(newArr, valObj)
	}

	if !found {
		newArr = append(newArr, PDFStringLiteral(key))
		newArr = append(newArr, val)
	}

	dict.Update("Names", newArr)

	logInfoTypes.Println("LeafNodeSetValue end")
	return
}

// LeafNodeRemoveValue removes a key/value pair that is assumed to live in leaf.
// found returns true on successful removal.
// deadLeaf is an indicator for having to remove leaf after leaf's last key/value pair gets deleted.
func (nameTree *PDFNameTree) LeafNodeRemoveValue(xRefTable *XRefTable, root bool, key string) (found, deadLeaf bool, err error) {

	logInfoTypes.Printf("LeafNodeRemoveValue begin: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	var larr *PDFArray

	if !root {

		var minKey, maxKey string
		larr, minKey, maxKey, err = nameTree.limitsArray(xRefTable)
		if err != nil {
			return
		}

		if key < minKey || key > maxKey {
			err = errors.Errorf("LeafNodeRemoveValue: key=%s corrupt leaf node: %v\n", key, nameTree)
			return
		}

	}

	var narr *PDFArray
	narr, err = nameTree.namesArray(xRefTable)
	if err != nil {
		return
	}

	a := *narr

	// Iterate over contents of Names array - a sequence of key value pairs.
	for i := 0; i < len(a)/2; i++ {

		var k string
		k, err = nameTree.key(xRefTable, a[i*2])
		v := a[i*2+1]

		logInfoTypes.Printf("LeafNodeRemoveValue: k=%s v=%v i=%d\n", k, v, i)

		if key == k {

			logInfoTypes.Println("LeafNodeRemoveValue: we have a match!")

			// Remove obj graph of key
			logInfoTypes.Println("LeafNodeRemoveValue: deleting object graph of k")
			err = xRefTable.DeleteObjectGraph(a[i*2])
			if err != nil {
				return
			}

			// Remove object graph of val
			logInfoTypes.Println("LeafNodeRemoveValue: deleting object graph of v")
			err = xRefTable.DeleteObjectGraph(v)
			if err != nil {
				return
			}

			logInfoTypes.Printf("LeafNodeRemoveValue: Names array=%v\n", a)
			// Remove key/value pair from narr
			if 2*i == len(a)-2 {
				a = a[:len(a)-2]
			} else {
				a = append(a[:2*i], a[2*i+2:]...)
			}

			logInfoTypes.Printf("LeafNodeRemoveValue: updated Names array=%v\n", a)

			dict, _ := xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
			dict.Update("Names", a)

			if len(a) == 0 {
				deadLeaf = true
			} else {
				// Update Limits
				if !root {
					(*larr)[0] = a[0]
					(*larr)[1] = a[len(a)-2]
					//dict.Update("Limits", a)
				}
			}

			found = true
			logInfoTypes.Println("LeafNodeRemoveValue end")
			return
		}

	}

	logInfoTypes.Println("LeafNodeRemoveValue end: not found")
	return
}

// Value returns the value for a given key.
func (nameTree PDFNameTree) Value(xRefTable *XRefTable, key string) (indRef *PDFIndirectRef, err error) {

	logInfoTypes.Printf("Value: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	var leafNode *PDFNameTree

	leafNode, err = nameTree.LeafNode(xRefTable, true, key)
	if err != nil {
		return
	}

	return leafNode.LeafNodeValue(xRefTable, key)
}

// SetValue add or updates a key value pair.
func (nameTree *PDFNameTree) SetValue(xRefTable *XRefTable, key string, val PDFIndirectRef) (err error) {

	logInfoTypes.Printf("SetValueValue: obj#%d key=%s val=%v\n", nameTree.rootObjNr(), key, val)

	var leafNode *PDFNameTree

	leafNode, err = nameTree.LeafNode(xRefTable, true, key)
	if err != nil {
		return
	}

	return leafNode.LeafNodeSetValue(xRefTable, key, val)
}

func (nameTree *PDFNameTree) removeKid(xRefTable *XRefTable, dict *PDFDict, arr *PDFArray, pos int) (deadKid bool, err error) {

	if len(*arr) > 1 {

		// Remove kid i
		a := *arr
		if pos == len(a)-1 {
			a = a[:pos]
		} else {
			a = append(a[:pos], a[:pos+1]...)
		}
		dict.Update("Kids", arr)

		// Update Limits after kid removal.
		var lFirst, lLast, l *PDFArray

		lFirst, _, _, err = NewNameTree(a[0].(PDFIndirectRef)).limitsArray(xRefTable)
		if err != nil {
			return
		}

		lLast, _, _, err = NewNameTree(a[len(a)-1].(PDFIndirectRef)).limitsArray(xRefTable)
		if err != nil {
			return
		}

		l, _, _, err = nameTree.limitsArray(xRefTable)
		if err != nil {
			return
		}

		// min = first kid limits min
		(*l)[0] = (*lFirst)[0]

		// max = last kid limits max
		(*l)[1] = (*lLast)[1]

		deadKid = false

	} else {
		err = xRefTable.DeleteObject(nameTree.rootObjNr())
		if err != nil {
			return
		}

		deadKid = true
	}

	return
}

func (nameTree *PDFNameTree) checkLimits(xRefTable *XRefTable, dict *PDFDict, key string) (skip, ok bool, err error) {

	var o interface{}
	var found bool

	o, found = dict.Find("Limits")
	if !found {
		return
	}

	var arr *PDFArray

	arr, err = xRefTable.DereferenceArray(o)
	if err != nil {
		return
	}

	var minKey, maxKey string
	minKey, maxKey, err = nameTree.limits(arr)
	if err != nil {
		return
	}

	if key < minKey {
		// name tree does not contain key, nothing removed.
		ok = true
		skip = true
		logInfoTypes.Println("RemoveKeyValuePair end: name does not contain key, nothing removed")
		return
	}

	// Skip this subtree
	if key > maxKey {
		skip = true
		logInfoTypes.Println("RemoveKeyValuePair end: skip this subtree")
		return
	}

	// We are in the correct subtree.
	return
}

// RemoveKeyValuePair removes a key value pair.
// ok is an indicator for stopping recursion.
// found returns true on successful removal.
// deadLeaf is an indicator for having to remove a kid after a leaf's last key/value pair got deleted.
func (nameTree *PDFNameTree) RemoveKeyValuePair(xRefTable *XRefTable, root bool, key string) (ok, found, deadKid bool, err error) {

	objNumber := nameTree.rootObjNr()

	logInfoTypes.Printf("RemoveKeyValuePair: obj#%d key=%s\n", objNumber, key)

	// A node has "Kids" or "Names" entry.

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return // ?
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, foundKids := dict.Find("Kids"); foundKids {

		// Check Limits array if present (intermediate nodes).
		var skip bool
		skip, ok, err = nameTree.checkLimits(xRefTable, dict, key)
		if err != nil || skip {
			return
		}

		// We are in the correct subtree.
		var arr *PDFArray
		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return
		}
		if arr == nil {
			err = errors.New("RemoveKeyValuePair: missing \"Kids\" array")
			return
		}

		for i, obj := range *arr {

			logInfoTypes.Printf("RemoveKeyValuePair: processing kid: %v\n", obj)

			var kid PDFIndirectRef
			kid, ok = obj.(PDFIndirectRef)
			if !ok {
				return false, false, false, errors.New("RemoveKeyValuePair: corrupt kid")
			}

			ok, found, deadKid, err = NewNameTree(kid).RemoveKeyValuePair(xRefTable, false, key)
			if err != nil {
				return
			}

			if !ok {
				continue
			}

			// operation completed.

			if found {

				// key found.

				logDebugTypes.Printf("RemoveKeyValuePair: key:%s found", key)

				if deadKid {

					// sole key removed, need to remove corresponding kid.
					logDebugTypes.Printf("RemoveKeyValuePair: deadKid, will remove kid %d from len=%d\n", i, len(*arr))

					deadKid, err = nameTree.removeKid(xRefTable, dict, arr, i)
					if err != nil {
						return
					}

					logInfoTypes.Println("RemoveKeyValuePair end")
					return
				}

			} else {
				// key not found in proper subtree, nothing removed.
				logDebugTypes.Printf("RemoveKeyValuePair: key:%s not found", key)
			}

			// Recursion stops here.
			logInfoTypes.Println("RemoveKeyValuePair end1")
			return
		}

		logInfoTypes.Println("RemoveKeyValuePair end")
		return
	}

	// Leaf node

	found, deadKid, err = nameTree.LeafNodeRemoveValue(xRefTable, root, key)

	ok = true

	logInfoTypes.Println("RemoveKeyValuePair end: leaf")

	return
}
