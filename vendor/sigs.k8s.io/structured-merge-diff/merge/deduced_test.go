/*
Copyright 2019 The Kubernetes Authors.

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

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

func TestDeduced(t *testing.T) {
	tests := map[string]TestCase{
		"leaf_apply_twice": {
			Ops: []Operation{
				Apply{
					Manager: "default",
					Object: `
						numeric: 1
						string: "string"
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "default",
					Object: `
						numeric: 2
						string: "string"
						bool: false
					`,
					APIVersion: "v1",
				},
			},
			Object: `
				numeric: 2
				string: "string"
				bool: false
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("numeric"), _P("string"), _P("bool"),
					),
					APIVersion: "v1",
				},
			},
		},
		"leaf_apply_update_apply_no_conflict": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "string"
					`,
				},
				Update{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "string"
						bool: true
					`,
				},
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 2
						string: "string"
					`,
				},
			},
			Object: `
				numeric: 2
				string: "string"
				bool: true
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("numeric"), _P("string"),
					),
					APIVersion: "v1",
				},
				"controller": &fieldpath.VersionedSet{
					Set: _NS(
						_P("bool"),
					),
					APIVersion: "v1",
				},
			},
		},
		"leaf_apply_update_apply_with_conflict": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "string"
					`,
				},
				Update{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "controller string"
						bool: true
					`,
				},
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 2
						string: "user string"
					`,
					Conflicts: merge.Conflicts{
						merge.Conflict{Manager: "controller", Path: _P("string")},
					},
				},
				ForceApply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 2
						string: "user string"
					`,
				},
			},
			Object: `
				numeric: 2
				string: "user string"
				bool: true
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("numeric"), _P("string"),
					),
					APIVersion: "v1",
				},
				"controller": &fieldpath.VersionedSet{
					Set: _NS(
						_P("bool"),
					),
					APIVersion: "v1",
				},
			},
		},
		"leaf_apply_twice_dangling": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "string"
						bool: false
					`,
				},
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						string: "new string"
					`,
				},
			},
			Object: `
				numeric: 1
				string: "new string"
				bool: false
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("string"),
					),
					APIVersion: "v1",
				},
			},
		},
		"leaf_update_remove_empty_set": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						string: "string"
					`,
				},
				Update{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						string: "new string"
					`,
				},
			},
			Object: `
				string: "new string"
			`,
			Managed: fieldpath.ManagedFields{
				"controller": &fieldpath.VersionedSet{
					Set: _NS(
						_P("string"),
					),
					APIVersion: "v1",
				},
			},
		},
		"apply_twice_list_is_atomic": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						list:
						- a
						- c
					`,
				},
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						list:
						- a
						- d
						- c
						- b
					`,
				},
			},
			Object: `
				list:
				- a
				- d
				- c
				- b
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set:        _NS(_P("list")),
					APIVersion: "v1",
				},
			},
		},
		"apply_update_apply_list": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						list:
						- a
						- c
					`,
				},
				Update{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						list:
						- a
						- b
						- c
						- d
					`,
				},
				ForceApply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						list:
						- a
						- b
						- c
					`,
				},
			},
			Object: `
				list:
				- a
				- b
				- c
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set:        _NS(_P("list")),
					APIVersion: "v1",
				},
			},
		},
		"leaf_apply_remove_empty_set": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						string: "string"
					`,
				},
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object:     ``,
				},
			},
			Object: `
				string: "string"
			`,
			Managed: fieldpath.ManagedFields{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(typed.DeducedParseableType{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
