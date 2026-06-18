// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Licensed under the Mozilla Public License, Version 2.0.
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package runtime

import (
	"lunex/internal/errfmt"
	"strings"
	"sync"
)

type Environment struct {
	vars    map[string]*Value
	consts  map[string]bool
	parent  *Environment
	escaped bool
}

// envPool recycles Environment objects to cut GC pressure in tight loops.
var envPool = sync.Pool{
	New: func() any {
		return &Environment{
			vars:   make(map[string]*Value, 8),
			consts: make(map[string]bool, 4),
		}
	},
}

func NewEnvironment(parent *Environment) *Environment {
	e := envPool.Get().(*Environment)
	// Clear maps without re-allocating (keep capacity, reset length).
	for k := range e.vars {
		delete(e.vars, k)
	}
	for k := range e.consts {
		delete(e.consts, k)
	}
	e.parent = parent
	return e
}

// ReleaseEnvironment returns an environment back to the pool.
// Call this only when you are sure no live reference to the environment
// (or its values) remains — typically at the end of a function call or
// block scope. Forgetting to call it is safe but wastes memory.
func ReleaseEnvironment(e *Environment) {
	if e == nil {
		return
	}
	if e.escaped {
		return
	}
	for k := range e.vars {
		delete(e.vars, k)
	}
	for k := range e.consts {
		delete(e.consts, k)
	}
	e.parent = nil
	envPool.Put(e)
}

// MarkEscaped marks this environment and all its ancestors as escaped
// (not eligible for pooling). Call this whenever a closure captures env.
func MarkEscaped(e *Environment) {
	for cur := e; cur != nil; cur = cur.parent {
		if cur.escaped {
			break
		}
		cur.escaped = true
	}
}

func (e *Environment) Define(name string, val *Value, isConst bool) {
	e.vars[name] = val
	if isConst {
		e.consts[name] = true
	}
}

func (e *Environment) SetLocal(name string, val *Value) {
	e.vars[name] = val
}

func (e *Environment) GetLocal(name string) (*Value, bool) {
	v, ok := e.vars[name]
	return v, ok
}

func (e *Environment) Set(name string, val *Value) error {
	if _, ok := e.vars[name]; ok {
		if e.consts[name] {
			return errfmt.ConstReassignError(name, "", 0, 0, nil)
		}
		e.vars[name] = val
		return nil
	}
	if e.parent != nil {
		if _, ok := e.parent.vars[name]; ok {
			if e.parent.consts[name] {
				return errfmt.ConstReassignError(name, "", 0, 0, nil)
			}
			e.parent.vars[name] = val
			return nil
		}
		if e.parent.parent != nil {
			env := e.parent.parent.find(name)
			if env != nil {
				if env.consts[name] {
					return errfmt.ConstReassignError(name, "", 0, 0, nil)
				}
				env.vars[name] = val
				return nil
			}
		}
	}
	return errfmt.ReferenceError(name, "", 0, 0, nil)
}

func (e *Environment) Get(name string) (*Value, bool) {
	if v, ok := e.vars[name]; ok {
		return v, true
	}
	if e.parent != nil {
		if v, ok := e.parent.vars[name]; ok {
			return v, true
		}
		if e.parent.parent != nil {
			env := e.parent.parent.find(name)
			if env != nil {
				return env.vars[name], true
			}
		}
	}
	return Undefined, false
}

func (e *Environment) find(name string) *Environment {
	cur := e
	for cur != nil {
		if _, ok := cur.vars[name]; ok {
			return cur
		}
		cur = cur.parent
	}
	return nil
}

func (e *Environment) Has(name string) bool {
	return e.find(name) != nil
}

func (e *Environment) Parent() *Environment {
	return e.parent
}

func (e *Environment) AllNames() []string {
	seen := make(map[string]bool)
	cur := e
	for cur != nil {
		for k := range cur.vars {
			if !strings.HasPrefix(k, "__") {
				seen[k] = true
			}
		}
		cur = cur.parent
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}
