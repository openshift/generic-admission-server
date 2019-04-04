/*
Copyright 2018 The Kubernetes Authors.

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

package fixture

import (
	"bytes"
	"fmt"
	"reflect"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// State of the current test in terms of live object. One can check at
// any time that Live and Managers match the expectations.
type State struct {
	Live     typed.TypedValue
	Parser   typed.ParseableType
	Managers fieldpath.ManagedFields
	Updater  *merge.Updater
}

// FixTabsOrDie counts the number of tab characters preceding the first line in
// the given yaml object. It removes that many tabs from every line. It then
// converts remaining tabs to spaces (two spaces per tab). It panics (it's a
// test funtion) if it finds mixed tabs and spaces in front of a line, or if
// some line has fewer tabs than the first line.
//
// The purpose of this is to make it easier to read tests.
func FixTabsOrDie(in typed.YAMLObject) typed.YAMLObject {
	consumeTabs := func(line []byte) (tabCount int, spacesFound bool) {
		for _, c := range line {
			if c == ' ' {
				spacesFound = true
			}
			if c != '\t' {
				break
			}
			tabCount++
		}
		return tabCount, spacesFound
	}

	lines := bytes.Split([]byte(in), []byte{'\n'})
	if len(lines[0]) == 0 && len(lines) > 1 {
		lines = lines[1:]
	}
	prefix, _ := consumeTabs(lines[0])
	var anySpacesFound bool
	var anyTabsFound bool

	for i := range lines {
		line := lines[i]
		indent, spacesFound := consumeTabs(line)
		if i == len(lines)-1 && len(line) <= prefix && indent == len(line) {
			// It's OK for the last line to be blank (trailing \n)
			lines[i] = []byte{}
			break
		}
		anySpacesFound = anySpacesFound || spacesFound
		anyTabsFound = anyTabsFound || indent > 0
		if indent < prefix {
			panic(fmt.Sprintf("line %v doesn't have %v tabs as a prefix:\n%s", i, prefix, in))
		}
		lines[i] = append(bytes.Repeat([]byte{' ', ' '}, indent-prefix), line[indent:]...)
	}
	if anyTabsFound && anySpacesFound {
		panic("mixed tabs and spaces found:\n" + string(in))
	}
	return typed.YAMLObject(bytes.Join(lines, []byte{'\n'}))
}

func (s *State) checkInit() error {
	if s.Live == nil {
		obj, err := s.Parser.FromYAML("{}")
		if err != nil {
			return fmt.Errorf("failed to create new empty object: %v", err)
		}
		s.Live = obj
	}
	return nil
}

// Update the current state with the passed in object
func (s *State) Update(obj typed.YAMLObject, version fieldpath.APIVersion, manager string) error {
	obj = FixTabsOrDie(obj)
	if err := s.checkInit(); err != nil {
		return err
	}
	tv, err := s.Parser.FromYAML(obj)
	managers, err := s.Updater.Update(s.Live, tv, version, s.Managers, manager)
	if err != nil {
		return err
	}
	s.Live = tv
	s.Managers = managers

	return nil
}

// Apply the passed in object to the current state
func (s *State) Apply(obj typed.YAMLObject, version fieldpath.APIVersion, manager string, force bool) error {
	obj = FixTabsOrDie(obj)
	if err := s.checkInit(); err != nil {
		return err
	}
	tv, err := s.Parser.FromYAML(obj)
	if err != nil {
		return err
	}
	new, managers, err := s.Updater.Apply(s.Live, tv, version, s.Managers, manager, force)
	if err != nil {
		return err
	}
	s.Live = new
	s.Managers = managers

	return nil
}

// CompareLive takes a YAML string and returns the comparison with the
// current live object or an error.
func (s *State) CompareLive(obj typed.YAMLObject) (*typed.Comparison, error) {
	obj = FixTabsOrDie(obj)
	if err := s.checkInit(); err != nil {
		return nil, err
	}
	tv, err := s.Parser.FromYAML(obj)
	if err != nil {
		return nil, err
	}
	return s.Live.Compare(tv)
}

// dummyConverter doesn't convert, it just returns the same exact object, as long as a version is provided.
type dummyConverter struct{}

// Convert returns the object given in input, not doing any conversion.
func (dummyConverter) Convert(v typed.TypedValue, version fieldpath.APIVersion) (typed.TypedValue, error) {
	if len(version) == 0 {
		return nil, fmt.Errorf("cannot convert to invalid version: %q", version)
	}
	return v, nil
}

// Operation is a step that will run when building a table-driven test.
type Operation interface {
	run(*State) error
}

// Apply is a type of operation. It is a non-forced apply run by a
// manager with a given object. Since non-forced apply operation can
// conflict, the user can specify the expected conflicts. If conflicts
// don't match, an error will occur.
type Apply struct {
	Manager    string
	APIVersion fieldpath.APIVersion
	Object     typed.YAMLObject
	Conflicts  merge.Conflicts
}

var _ Operation = &Apply{}

func (a Apply) run(state *State) error {
	err := state.Apply(a.Object, a.APIVersion, a.Manager, false)
	if (err != nil || a.Conflicts != nil) && !reflect.DeepEqual(err, a.Conflicts) {
		return fmt.Errorf("expected conflicts: %v, got %v", a.Conflicts, err)
	}
	return nil

}

// ForceApply is a type of operation. It is a forced-apply run by a
// manager with a given object. Any error will be returned.
type ForceApply struct {
	Manager    string
	APIVersion fieldpath.APIVersion
	Object     typed.YAMLObject
}

var _ Operation = &ForceApply{}

func (f ForceApply) run(state *State) error {
	return state.Apply(f.Object, f.APIVersion, f.Manager, true)
}

// Update is a type of operation. It is a controller type of
// update. Errors are passed along.
type Update struct {
	Manager    string
	APIVersion fieldpath.APIVersion
	Object     typed.YAMLObject
}

var _ Operation = &Update{}

func (u Update) run(state *State) error {
	return state.Update(u.Object, u.APIVersion, u.Manager)
}

// TestCase is the list of operations that need to be run, as well as
// the object/managedfields as they are supposed to look like after all
// the operations have been successfully performed. If Object/Managed is
// not specified, then the comparison is not performed (any object or
// managed field will pass). Any error (conflicts aside) happen while
// running the operation, that error will be returned right away.
type TestCase struct {
	// Ops is the list of operations to run sequentially
	Ops []Operation
	// Object, if not empty, is the object as it's expected to
	// be after all the operations are run.
	Object typed.YAMLObject
	// Managed, if not nil, is the ManagedFields as expected
	// after all operations are run.
	Managed fieldpath.ManagedFields
}

// Test runs the test-case using the given parser.
func (tc TestCase) Test(parser typed.ParseableType) error {
	state := State{
		Updater: &merge.Updater{Converter: &dummyConverter{}},
		Parser:  parser,
	}
	// We currently don't have any test that converts, we can take
	// care of that later.
	for i, ops := range tc.Ops {
		err := ops.run(&state)
		if err != nil {
			return fmt.Errorf("failed operation %d: %v", i, err)
		}
	}

	// If LastObject was specified, compare it with LiveState
	if tc.Object != typed.YAMLObject("") {
		comparison, err := state.CompareLive(tc.Object)
		if err != nil {
			return fmt.Errorf("failed to compare live with config: %v", err)
		}
		if !comparison.IsSame() {
			return fmt.Errorf("expected live and config to be the same:\n%v", comparison)
		}
	}

	if tc.Managed != nil {
		if diff := state.Managers.Difference(tc.Managed); len(diff) != 0 {
			return fmt.Errorf("expected Managers to be %v, got %v", tc.Managed, state.Managers)
		}
	}

	// Fail if any empty sets are present in the managers
	for manager, set := range state.Managers {
		if set.Empty() {
			return fmt.Errorf("expected Managers to have no empty sets, but found one managed by %v", manager)
		}
	}

	return nil
}
