// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package rfc7951

import (
	"strings"

	"github.com/danos/encoding/rfc7951/data"
	"github.com/danos/yang/schema"
)

type RFC7951Merger struct {
	schema schema.ModelSet
	tree   *data.Tree
}

func NewRFC7951Merger(sch schema.ModelSet, tree *data.Tree) *RFC7951Merger {
	return &RFC7951Merger{
		schema: sch,
		tree:   tree,
	}
}

func (m *RFC7951Merger) Tree() *data.Tree {
	return m.tree
}

func (m *RFC7951Merger) Merge(other *data.Tree) {
	m.tree = data.TreeFromObject(
		m.merge(m.schema, m.tree.Root(), other.Root()).
			AsObject())
}

func (m *RFC7951Merger) merge(
	sn schema.Node,
	this, new *data.Value,
) *data.Value {
	return this.Perform(
		func(arr *data.Array) *data.Value {
			return m.mergeArray(sn, arr, new)
		},
		func(obj *data.Object) *data.Value {
			return m.mergeObject(sn, obj, new)
		},
		func(v *data.Value) *data.Value {
			return new
		},
	).(*data.Value)
}

func (m *RFC7951Merger) mergeObject(
	sn schema.Node,
	obj *data.Object,
	new *data.Value,
) *data.Value {
	return new.Perform(func(n *data.Object) *data.Value {
		out := obj.Transform(func(out *data.TObject) {
			obj.Range(func(key string, val *data.Value) {
				if n.Contains(key) {
					sChild := sn.Child(m.parseKey(key))
					out = out.Assoc(key,
						m.merge(sChild,
							val, n.At(key)))
				}
			})
			n.Range(func(key string, val *data.Value) {
				if !obj.Contains(key) {
					out = out.Assoc(key, val)
				}
			})
		})
		return data.ValueNew(out)
	}, func(_ interface{}) *data.Value {
		// By default just return the original object; can't merge
		// unlike types.
		return data.ValueNew(obj)
	}).(*data.Value)
}

func (m *RFC7951Merger) mergeArray(
	sn schema.Node,
	arr *data.Array,
	new *data.Value,
) *data.Value {
	list, isList := sn.(schema.List)
	if isList {
		return m.mergeArrayByKey(list, arr, new)
	}
	return new.Perform(func(n *data.Array) *data.Value {
		out := arr.Transform(func(out *data.TArray) {
			arr.Range(func(i int, v *data.Value) {
				if n.Contains(i) {
					out = out.Assoc(i,
						m.merge(sn, v, n.At(i)))
				}
			})
			n.Range(func(i int, v *data.Value) {
				if !arr.Contains(i) {
					out = out.Append(v)
				}
			})
		})
		return data.ValueNew(out)
	}, func(_ interface{}) *data.Value {
		// By default just return the original array; can't merge
		// unlike types.
		return data.ValueNew(arr)
	}).(*data.Value)
}

// mergeArrayByKey will treat a List like an object when merging.
// it does this by first resolving the keys of the objects to indicies
// in the array then merging by key instead of index. Any new entries
// will be appened to the end of the array.
func (m *RFC7951Merger) mergeArrayByKey(
	list schema.List,
	arr *data.Array,
	new *data.Value,
) *data.Value {
	keys := list.Keys()
	return new.Perform(func(n *data.Array) *data.Value {
		out := arr.Transform(func(out *data.TArray) {
			entries := m.entriesFromArray(keys, arr)
			newEntries := m.entriesFromArray(keys, n)
			for k, i := range entries {
				val := arr.At(i)
				new, ok := newEntries[k]
				sChild := list.Child(k)
				if ok {
					out = out.Assoc(i,
						m.merge(sChild, val, n.At(new)))
				}
			}
			for k, i := range newEntries {
				val := n.At(i)
				_, ok := entries[k]
				if !ok {
					out = out.Append(val)
				}
			}
		})
		return data.ValueNew(out)
	}, func(_ interface{}) *data.Value {
		// By default just return the original array; can't merge
		// unlike types.
		return data.ValueNew(arr)
	}).(*data.Value)
}

// entriesFromArray will create a map from the object key to its
// index in the array. These entries will be used for random access
// into the array during the merge.
func (m *RFC7951Merger) entriesFromArray(
	keys []string,
	arr *data.Array,
) map[string]int {
	entries := make(map[string]int)
	arr.Range(func(i int, v *data.Value) {
		obj := v.ToObject()
		if obj == nil {
			return //skip malformed data
		}
		// The key that is tracked is the concatenation of all
		// the key values with Middle Dot ("·") as a
		// seperator.
		keyValues := make([]string, len(keys))
		for i, k := range keys {
			val := obj.At(k)
			if val == nil {
				return //skip malformed data
			}
			keyValues[i] = val.String()
		}
		entries[strings.Join(keyValues, "·")] = i
	})
	return entries
}

// parseKey returns on the moduleless version of a key. Schema nodes
// are not addressable by module prefixed names so we need to strip it
// away
func (m *RFC7951Merger) parseKey(in string) string {
	elems := strings.SplitN(in, ":", 2)
	switch len(elems) {
	case 1:
		return elems[0]
	default:
		return elems[1]
	}
}
