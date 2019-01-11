/*
Copyright (c) 2017-2018 VMware, Inc. All Rights Reserved.

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

package simulator

import (
	"fmt"
	"reflect"

	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// Assigns b to a (types must match for assignement) if b is non zero-value for its type.
// If types are struct - recurses through fields and sets non zero-value b fields to a.
func assignNonZeroValue(a, b interface{}) error {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	if va.Kind() != reflect.Ptr {
		return fmt.Errorf("a type must be Ptr")
	}

	va = reflect.Indirect(va)
	// Types must be equal for a to be set to b
	if va.Type() != vb.Type() {
		return fmt.Errorf("a type: %v should be equal b type: %v", va.Type(), vb.Type())
	}

	assignNonZeroValueRecursive(va, vb)
	return nil
}

func assignNonZeroValueRecursive(a, b reflect.Value) {
	switch b.Kind() {
	case reflect.Struct:
		for i := 0; i < b.NumField(); i++ {
			assignNonZeroValueRecursive(a.Field(i), b.Field(i))
		}
	default:
		// Check if b is zero-value of its underlying type
		bi := b.Interface()
		isZero := reflect.DeepEqual(bi, reflect.Zero(reflect.TypeOf(bi)).Interface())

		// Only assign if b is not zero-value
		if !isZero && a.CanSet() {
			a.Set(b)
		}
	}
}

func SetCustomValue(ctx *Context, req *types.SetCustomValue) soap.HasFault {
	ctx.Caller = &req.This
	body := &methods.SetCustomValueBody{}

	cfm := Map.CustomFieldsManager()

	_, field := cfm.findByNameType(req.Key, req.This.Type)
	if field == nil {
		body.Fault_ = Fault("", &types.InvalidArgument{InvalidProperty: "key"})
		return body
	}

	res := cfm.SetField(ctx, &types.SetField{
		This:   cfm.Reference(),
		Entity: req.This,
		Key:    field.Key,
		Value:  req.Value,
	})

	if res.Fault() != nil {
		body.Fault_ = res.Fault()
		return body
	}

	body.Res = &types.SetCustomValueResponse{}
	return body
}
