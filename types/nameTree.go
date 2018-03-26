package types

import (
	"github.com/hhrutter/pdfcpu/log"
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

func (nameTree PDFNameTree) namesArray(xRefTable *XRefTable) (*PDFArray, error) {

	dict, err := xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return nil, err
	}

	obj, ok := dict.Find("Names")
	if !ok {
		return nil, errors.Errorf("namesArray: missing \"Names\" entry in <%v>\n", obj)
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return nil, err
	}

	if len(*arr)%2 > 0 {
		return nil, errors.Errorf("limitsArray: corrupt \"Names\" entry in %v\n", *arr)
	}

	return arr, nil
}

func (nameTree PDFNameTree) limits(arr *PDFArray) (min, max string, err error) {

	if len(*arr) != 2 {
		return "", "", errors.Errorf("limits: corrupt \"Limits\" entry in %v\n", *arr)
	}

	sl, ok := (*arr)[0].(PDFStringLiteral)
	if !ok {
		err = errors.Errorf("limits: corrupt min key <%v>\n", (*arr)[0])
		return "", "", err
	}
	min, err = StringLiteralToString(sl.Value())
	if err != nil {
		return "", "", err
	}

	sl, ok = (*arr)[1].(PDFStringLiteral)
	if !ok {
		return "", "", errors.Errorf("limits: corrupt max key <%v>\n", (*arr)[1])
	}

	max, err = StringLiteralToString(sl.Value())
	if err != nil {
		return "", "", err
	}

	return min, max, nil
}

func (nameTree PDFNameTree) limitsArray(xRefTable *XRefTable) (arr *PDFArray, min, max string, err error) {

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return nil, "", "", err
	}

	obj, ok := dict.Find("Limits")
	if !ok {
		return nil, "", "", errors.Errorf("limitsArray: missing \"Limits\" entry in <%v>\n", dict)
	}

	arr, err = xRefTable.DereferenceArray(obj)
	if err != nil {
		return nil, "", "", err
	}

	min, max, err = nameTree.limits(arr)

	return arr, min, max, err
}

// LeafNode retrieves the leaf node dict for a given key which may be a new one.
// last is true for the right most kid assuming strict left to right tree traversal.
func (nameTree *PDFNameTree) LeafNode(xRefTable *XRefTable, last bool, key string) (leaf *PDFNameTree, err error) {

	log.Debug.Printf("LeafNode: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	// A node has "Kids" or "Names" entry.

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return nil, err
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, found := dict.Find("Kids"); found {

		var arr *PDFArray

		// Check Limits array if present (intermediate nodes).
		if o, found := dict.Find("Limits"); found {

			arr, err = xRefTable.DereferenceArray(o)
			if err != nil {
				return nil, err
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
			return nil, err
		}
		if arr == nil {
			return nil, errors.New("LeafNode: missing \"Kids\" array")
		}

		kidCount := len(*arr)

		for i, obj := range *arr {

			log.Debug.Printf("LeafNode: processing kid: %v\n", obj)

			kid, ok := obj.(PDFIndirectRef)
			if !ok {
				return nil, errors.New("LeafNode: corrupt kid")
			}

			leaf, err = NewNameTree(kid).LeafNode(xRefTable, i == kidCount-1, key)
			if err != nil {
				return nil, err
			}
			if leaf != nil {
				return leaf, nil
			}

		}

		log.Debug.Println("LeafNode end")

		return nil, nil
	}

	// Leaf node
	leaf = nameTree

	log.Debug.Println("LeafNode end")

	return leaf, nil
}

func (nameTree *PDFNameTree) key(xRefTable *XRefTable, o PDFObject) (string, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", err
	}

	sl, ok := o.(PDFStringLiteral)
	if !ok {
		return "", errors.Errorf("corrupt key <%v>\n", o)
	}

	return StringLiteralToString(sl.Value())
}

// LeafNodeValue retrieves the indRef value for a given key of a leaf node.
// Will return nil if key not found.
func (nameTree *PDFNameTree) LeafNodeValue(xRefTable *XRefTable, key string) (*PDFIndirectRef, error) {

	log.Debug.Printf("LeafNodeValue: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	arr, err := nameTree.namesArray(xRefTable)
	if err != nil {
		return nil, err
	}

	log.Debug.Printf("arr = %v\n", arr)

	var getValue bool

	for i, obj := range *arr {

		if i%2 == 0 {

			//var k string
			k, err := nameTree.key(xRefTable, obj)
			if err != nil {
				return nil, err
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
			log.Debug.Println("LeafNodeValue end")
			return &iRef, nil
		}

	}

	log.Debug.Println("LeafNodeValue end: not found")
	return nil, nil
}

// LeafNodeSetValue adds or updates a key value pair.
func (nameTree *PDFNameTree) LeafNodeSetValue(xRefTable *XRefTable, key string, val PDFIndirectRef) error {

	log.Debug.Printf("LeafNodeSetValue: obj#%d key=%s val=%v\n", nameTree.rootObjNr(), key, val)

	dict, err := xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return err
	}

	obj, ok := dict.Find("Names")
	if !ok {
		return errors.Errorf("LeafNodeSetValue: missing \"Names\" entry in <%v>\n", obj)
	}

	arr, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return err
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
			return err
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
					return err
				}

				// Free up all objs referred by val.
				err = xRefTable.DeleteObjectGraph(valObj)
				if err != nil {
					return err
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

	log.Debug.Println("LeafNodeSetValue end")
	return nil
}

// LeafNodeRemoveValue removes a key/value pair that is assumed to live in leaf.
// found returns true on successful removal.
// deadLeaf is an indicator for having to remove leaf after leaf's last key/value pair gets deleted.
func (nameTree *PDFNameTree) LeafNodeRemoveValue(xRefTable *XRefTable, root bool, key string) (found, deadLeaf bool, err error) {

	log.Debug.Printf("LeafNodeRemoveValue begin: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	var larr *PDFArray

	if !root {

		var minKey, maxKey string
		larr, minKey, maxKey, err = nameTree.limitsArray(xRefTable)
		if err != nil {
			return false, false, err
		}

		if key < minKey || key > maxKey {
			return false, false, errors.Errorf("LeafNodeRemoveValue: key=%s corrupt leaf node: %v\n", key, nameTree)
		}

	}

	narr, err := nameTree.namesArray(xRefTable)
	if err != nil {
		return false, false, err
	}

	a := *narr

	// Iterate over contents of Names array - a sequence of key value pairs.
	for i := 0; i < len(a)/2; i++ {

		var k string
		k, err = nameTree.key(xRefTable, a[i*2])
		v := a[i*2+1]

		log.Debug.Printf("LeafNodeRemoveValue: k=%s v=%v i=%d\n", k, v, i)

		if key == k {

			log.Debug.Println("LeafNodeRemoveValue: we have a match!")

			// Remove obj graph of key
			log.Debug.Println("LeafNodeRemoveValue: deleting object graph of k")
			err = xRefTable.DeleteObjectGraph(a[i*2])
			if err != nil {
				return false, false, err
			}

			// Remove object graph of val
			log.Debug.Println("LeafNodeRemoveValue: deleting object graph of v")
			err = xRefTable.DeleteObjectGraph(v)
			if err != nil {
				return false, false, err
			}

			log.Debug.Printf("LeafNodeRemoveValue: Names array=%v\n", a)
			// Remove key/value pair from narr
			if 2*i == len(a)-2 {
				a = a[:len(a)-2]
			} else {
				a = append(a[:2*i], a[2*i+2:]...)
			}

			log.Debug.Printf("LeafNodeRemoveValue: updated Names array=%v\n", a)

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
			log.Debug.Println("LeafNodeRemoveValue end")
			return found, deadLeaf, nil
		}

	}

	log.Debug.Println("LeafNodeRemoveValue end: not found")
	return false, false, nil
}

// Value returns the value for a given key.
func (nameTree PDFNameTree) Value(xRefTable *XRefTable, key string) (*PDFIndirectRef, error) {

	log.Debug.Printf("Value: obj#%d key=%s\n", nameTree.rootObjNr(), key)

	var leafNode *PDFNameTree

	leafNode, err := nameTree.LeafNode(xRefTable, true, key)
	if err != nil {
		return nil, err
	}

	return leafNode.LeafNodeValue(xRefTable, key)
}

// SetValue add or updates a key value pair.
func (nameTree *PDFNameTree) SetValue(xRefTable *XRefTable, key string, val PDFIndirectRef) error {

	log.Debug.Printf("SetValueValue: obj#%d key=%s val=%v\n", nameTree.rootObjNr(), key, val)

	var leafNode *PDFNameTree

	leafNode, err := nameTree.LeafNode(xRefTable, true, key)
	if err != nil {
		return err
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
			return false, err
		}

		lLast, _, _, err = NewNameTree(a[len(a)-1].(PDFIndirectRef)).limitsArray(xRefTable)
		if err != nil {
			return false, err
		}

		l, _, _, err = nameTree.limitsArray(xRefTable)
		if err != nil {
			return false, err
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

	return deadKid, nil
}

func (nameTree *PDFNameTree) checkLimits(xRefTable *XRefTable, dict *PDFDict, key string) (skip, ok bool, err error) {

	o, found := dict.Find("Limits")
	if !found {
		return false, false, nil
	}

	arr, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return false, false, err
	}

	var minKey, maxKey string
	minKey, maxKey, err = nameTree.limits(arr)
	if err != nil {
		return false, false, err
	}

	if key < minKey {
		// name tree does not contain key, nothing removed.
		ok = true
		skip = true
		log.Debug.Println("RemoveKeyValuePair end: name does not contain key, nothing removed")
		return skip, ok, nil
	}

	// Skip this subtree
	if key > maxKey {
		skip = true
		log.Debug.Println("RemoveKeyValuePair end: skip this subtree")
		return skip, false, nil
	}

	// We are in the correct subtree.
	return false, false, nil
}

// RemoveKeyValuePair removes a key value pair.
// ok is an indicator for stopping recursion.
// found returns true on successful removal.
// deadLeaf is an indicator for having to remove a kid after a leaf's last key/value pair got deleted.
func (nameTree *PDFNameTree) RemoveKeyValuePair(xRefTable *XRefTable, root bool, key string) (ok, found, deadKid bool, err error) {

	objNumber := nameTree.rootObjNr()

	log.Debug.Printf("RemoveKeyValuePair: obj#%d key=%s\n", objNumber, key)

	// A node has "Kids" or "Names" entry.

	var dict *PDFDict
	dict, err = xRefTable.DereferenceDict(nameTree.PDFIndirectRef)
	if err != nil || dict == nil {
		return false, false, false, err
	}

	// Kids: array of indirect references to the immediate children of this node.
	// if Kids present then recurse
	if obj, foundKids := dict.Find("Kids"); foundKids {

		// Check Limits array if present (intermediate nodes).
		var skip bool
		skip, ok, err = nameTree.checkLimits(xRefTable, dict, key)
		if err != nil || skip {
			return ok, false, false, err
		}

		// We are in the correct subtree.
		var arr *PDFArray
		arr, err = xRefTable.DereferenceArray(obj)
		if err != nil {
			return false, false, false, err
		}
		if arr == nil {
			return false, false, false, errors.New("RemoveKeyValuePair: missing \"Kids\" array")
		}

		for i, obj := range *arr {

			log.Debug.Printf("RemoveKeyValuePair: processing kid: %v\n", obj)

			var kid PDFIndirectRef
			kid, ok = obj.(PDFIndirectRef)
			if !ok {
				return false, false, false, errors.New("RemoveKeyValuePair: corrupt kid")
			}

			ok, found, deadKid, err = NewNameTree(kid).RemoveKeyValuePair(xRefTable, false, key)
			if err != nil {
				return false, false, false, err
			}

			if !ok {
				continue
			}

			// operation completed.

			if found {

				// key found.

				log.Debug.Printf("RemoveKeyValuePair: key:%s found", key)

				if deadKid {

					// sole key removed, need to remove corresponding kid.
					log.Debug.Printf("RemoveKeyValuePair: deadKid, will remove kid %d from len=%d\n", i, len(*arr))

					deadKid, err = nameTree.removeKid(xRefTable, dict, arr, i)
					if err != nil {
						return false, false, false, err
					}

					log.Debug.Println("RemoveKeyValuePair end")
					return ok, found, deadKid, nil
				}

			} else {
				// key not found in proper subtree, nothing removed.
				log.Debug.Printf("RemoveKeyValuePair: key:%s not found", key)
			}

			// Recursion stops here.
			log.Debug.Println("RemoveKeyValuePair end1")
			return ok, found, deadKid, nil
		}

		log.Debug.Println("RemoveKeyValuePair end")
		return ok, found, deadKid, nil
	}

	// Leaf node

	found, deadKid, err = nameTree.LeafNodeRemoveValue(xRefTable, root, key)

	ok = true

	log.Debug.Println("RemoveKeyValuePair end: leaf")

	return ok, found, deadKid, err
}
