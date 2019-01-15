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
	"strings"

	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// Function takes 2 objects and tries to assign src to dst.
// Assignement only works if object types are equal (or src points to value of dst type).
// If object types are struct - loops through all src fields (including nested structs fields)
// and assigns non-empty values to dst struct. Value is considered empty if:
//   1) It is nil (only for Ptr types);
//   2) It is set to its default value (only for struct fields with 'omitempty' tag).
func assignNonEmpty(dst, src interface{}) error {
	if src == nil {
		return nil
	}

	vdst := reflect.ValueOf(dst)
	vsrc := reflect.ValueOf(src)

	if vdst.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be Ptr")
	}

	vdst = reflect.Indirect(vdst)
	// src might point to value of dst type
	if vdst.Kind() != reflect.Ptr && vsrc.Kind() == reflect.Ptr {
		vsrc = reflect.Indirect(vsrc)
		if !vsrc.IsValid() {
			return nil
		}
	}

	// Types must be equal for dst to be set to src
	if vdst.Type() != vsrc.Type() {
		return fmt.Errorf("dst type: %v must be equal to src type: %v", vdst.Type(), vsrc.Type())
	}

	assignNonEmptyRecursive(vdst, vsrc)
	return nil
}

func assignNonEmptyRecursive(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Ptr:
		vsrc := src.Elem()
		if !vsrc.IsValid() {
			return
		}

		// If field was not set - allocate a new object
		if dst.IsNil() {
			dst.Set(reflect.New(vsrc.Type()))
		}
		assignNonEmptyRecursive(dst.Elem(), vsrc)
	case reflect.Struct:
		for i := 0; i < src.NumField(); i++ {
			srcf := src.Field(i)
			srci := srcf.Interface()

			// Check if src is zero-value of its underlying type
			isZero := reflect.DeepEqual(srci, reflect.Zero(reflect.TypeOf(srci)).Interface())
			// Check if src can be empty - struct field contains 'omitempty' tag
			canBeEmpty := strings.Contains(string(reflect.TypeOf(src.Interface()).Field(i).Tag), "omitempty")

			// If field is zero-value and has omitempty tag - it's empty and should not be used.
			// Every other time field is empty on purpose and should be used.
			if isZero && canBeEmpty {
				continue
			}

			assignNonEmptyRecursive(dst.Field(i), srcf)
		}
	default:
		if dst.CanSet() {
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
