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

// Assigns src to dst (types must match for assignement) if src is non zero-value for its type.
// If types are struct - recurses through fields and sets non zero-value src fields to dst.
func assignNonZeroValue(dst, src interface{}) error {
	if src == nil {
		return nil
	}

	vdst := reflect.ValueOf(dst)
	vsrc := reflect.ValueOf(src)

	if vdst.Kind() != reflect.Ptr {
		return fmt.Errorf("dst type must be Ptr")
	}

	vdst = reflect.Indirect(vdst)
	fmt.Printf("dst: %v - %v - %v\n", reflect.TypeOf(dst), dst, vdst)
	fmt.Printf("src: %v - %v - %v\n", reflect.TypeOf(src), src, vsrc)
	// Types must be equal for a to be set to b
	if vdst.Type() != vsrc.Type() {
		return fmt.Errorf("dst type: %v should be equal src type: %v", vdst.Type(), vsrc.Type())
	}

	assignNonZeroValueRecursive(vdst, vsrc)
	return nil
}

func assignNonZeroValueRecursive(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Ptr:
		vsrc := src.Elem()
		if !vsrc.IsValid() {
			return
		}
		assignNonZeroValueRecursive(dst.Elem(), vsrc)
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			assignNonZeroValueRecursive(dst.Field(i), src.Field(i))
		}
	default:
		// Check if b is zero-value of its underlying type
		srci := src.Interface()
		isZero := reflect.DeepEqual(srci, reflect.Zero(reflect.TypeOf(srci)).Interface())

		// Only assign if b is not zero-value
		if !isZero && dst.CanSet() {
			dst.Set(src)
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
