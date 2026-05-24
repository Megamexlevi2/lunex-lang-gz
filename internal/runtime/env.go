// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. Lunex Source License — attribution required, copying prohibited.
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package runtime

import (
        "fmt"
        "strings"
)

type Environment struct {
        vars   map[string]*Value
        consts map[string]bool
        parent *Environment
}

func NewEnvironment(parent *Environment) *Environment {
        return &Environment{
                vars:   make(map[string]*Value, 8),
                consts: make(map[string]bool),
                parent: parent,
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
                        return fmt.Errorf("TypeError: cannot reassign constant '%s' — use 'var' instead of 'val' if you need a mutable variable", name)
                }
                e.vars[name] = val
                return nil
        }
        if e.parent != nil {
                if _, ok := e.parent.vars[name]; ok {
                        if e.parent.consts[name] {
                                return fmt.Errorf("TypeError: cannot reassign constant '%s' — use 'var' instead of 'val' if you need a mutable variable", name)
                        }
                        e.parent.vars[name] = val
                        return nil
                }
                if e.parent.parent != nil {
                        env := e.parent.parent.find(name)
                        if env != nil {
                                if env.consts[name] {
                                        return fmt.Errorf("TypeError: cannot reassign constant '%s' — use 'var' instead of 'val' if you need a mutable variable", name)
                                }
                                env.vars[name] = val
                                return nil
                        }
                }
        }
        return fmt.Errorf("ReferenceError: '%s' is not defined — declare it with 'var' or 'val' first", name)
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
        if _, ok := e.vars[name]; ok {
                return e
        }
        if e.parent != nil {
                return e.parent.find(name)
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
        var collect func(env *Environment)
        collect = func(env *Environment) {
                if env == nil {
                        return
                }
                for k := range env.vars {
                        if !strings.HasPrefix(k, "__") {
                                seen[k] = true
                        }
                }
                collect(env.parent)
        }
        collect(e)
        names := make([]string, 0, len(seen))
        for k := range seen {
                names = append(names, k)
        }
        return names
}
