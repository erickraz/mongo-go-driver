// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bson

import (
	"bytes"
	"reflect"

	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// Unmarshaler is an interface implemented by types that can unmarshal a BSON
// document representation of themselves. The BSON bytes can be assumed to be
// valid. UnmarshalBSON must copy the BSON bytes if it wishes to retain the data
// after returning.
type Unmarshaler interface {
	UnmarshalBSON([]byte) error
}

// ValueUnmarshaler is an interface implemented by types that can unmarshal a
// BSON value representation of themselves. The BSON bytes and type can be
// assumed to be valid. UnmarshalBSONValue must copy the BSON value bytes if it
// wishes to retain the data after returning.
type ValueUnmarshaler interface {
	UnmarshalBSONValue(bsontype.Type, []byte) error
}

// Unmarshal parses the BSON-encoded data and stores the result in the value
// pointed to by val. If val is nil or not a pointer, Unmarshal returns
// InvalidUnmarshalError.
func Unmarshal(data []byte, val interface{}) error {
	return UnmarshalWithRegistry(DefaultRegistry, data, val)
}

// UnmarshalWithRegistry parses the BSON-encoded data using Registry r and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalWithRegistry(r *bsoncodec.Registry, data []byte, val interface{}) error {
	vr := bsonrw.NewBSONDocumentReader(data)
	return unmarshalFromReader(bsoncodec.DecodeContext{Registry: r}, vr, val)
}

// UnmarshalWithContext parses the BSON-encoded data using DecodeContext dc and
// stores the result in the value pointed to by val. If val is nil or not
// a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalWithContext(dc bsoncodec.DecodeContext, data []byte, val interface{}) error {
	vr := bsonrw.NewBSONDocumentReader(data)
	return unmarshalFromReader(dc, vr, val)
}

// UnmarshalExtJSON parses the extended JSON-encoded data and stores the result
// in the value pointed to by val. If val is nil or not a pointer, Unmarshal
// returns InvalidUnmarshalError.
func UnmarshalExtJSON(data []byte, canonical bool, val interface{}) error {
	return UnmarshalExtJSONWithRegistry(DefaultRegistry, data, canonical, val)
}

// UnmarshalExtJSONWithRegistry parses the extended JSON-encoded data using
// Registry r and stores the result in the value pointed to by val. If val is
// nil or not a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalExtJSONWithRegistry(r *bsoncodec.Registry, data []byte, canonical bool, val interface{}) error {
	ejvr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader(data), canonical)
	if err != nil {
		return err
	}

	return unmarshalFromReader(bsoncodec.DecodeContext{Registry: r}, ejvr, val)
}

// UnmarshalExtJSONWithContext parses the extended JSON-encoded data using
// DecodeContext dc and stores the result in the value pointed to by val. If val is
// nil or not a pointer, UnmarshalWithRegistry returns InvalidUnmarshalError.
func UnmarshalExtJSONWithContext(dc bsoncodec.DecodeContext, data []byte, canonical bool, val interface{}) error {
	ejvr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader(data), canonical)
	if err != nil {
		return err
	}

	return unmarshalFromReader(dc, ejvr, val)
}

func unmarshalFromReader(dc bsoncodec.DecodeContext, vr bsonrw.ValueReader, val interface{}) error {
	dec := decPool.Get().(*Decoder)
	defer decPool.Put(dec)

	err := dec.Reset(vr)
	if err != nil {
		return err
	}
	err = dec.SetContext(dc)
	if err != nil {
		return err
	}

	return dec.Decode(val)
}

// UnmarshalExtJSONWithRes reads in a val struct, and a res primitive.M (or []primitive.M)
// Warning: use with caution, using with custom unmarshaler is a bit tricky
// it is kind of hack since we changed some decode functions in mongo driver
// useful when we want to decode json into struct and have a bson map as byproduct

// ex:
// type t struct {
// 	A string
// 	B struct {
// 		B int
// 	}
// }

// tmp := t{}
// res := primitive.M{}
// err := bson.UnmarshalExtJSONWithRes([]byte(`{"a": "123", "b": {"b": 123}}`), true, &tmp, &res)
// {123 {true}} // tmp
// map[a:123 b:map[b:true]] // res

// note: multiple level of map might not work as expected

func UnmarshalExtJSONWithRes(data []byte, canonical bool, val, res interface{}) error {
	ejvr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader(data), canonical)
	if err != nil {
		return err
	}

	dec := decPool.Get().(*Decoder)
	defer decPool.Put(dec)

	err = dec.Reset(ejvr)
	if err != nil {
		return err
	}
	var vM = new(reflect.Value)

	err = dec.SetContext(bsoncodec.DecodeContext{Registry: DefaultRegistry, ValM: vM, SetVal: true})
	if err != nil {
		return err
	}
	// TODO: check val being struct
	// TODO: validate res being primitive.M or []primitive.M
	err = dec.Decode(val)
	if err != nil {
		return err
	}
	reflect.ValueOf(res).Elem().Set(*vM)

	return err
}
