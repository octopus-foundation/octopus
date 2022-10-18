/*
               .'\   /`.
             .'.-.`-'.-.`.
        ..._:   .-. .-.   :_...
      .'    '-.(o ) (o ).-'    `.
     :  _    _ _`~(_)~`_ _    _  :
    :  /:   ' .-=_   _=-. `   ;\  :
    :   :|-.._  '     `  _..-|:   :
     :   `:| |`:-:-.-:-:'| |:'   :
      `.   `.| | | | | | |.'   .'
        `.   `-:_| | |_:-'   .'
          `-._   ````    _.-'
              ``-------''

Created by ab, 03.10.2022
*/

package types

import (
	"testing"
)

func TestScopedName_ToParent(t *testing.T) {
	// a.b.c.D
	// a.b.D
	// a.D
	// D
	name := ScopedName{
		name:     "D",
		parent:   []string{"a", "b", "c"},
		fullPath: "a.b.c.D",
	}

	parent := name.ToParent()
	if parent.fullPath != "a.b.c" {
		t.Fatalf("Wrong parent path: %v", parent.fullPath)
	}

	parent = parent.ToParent()
	if parent.fullPath != "a.b" {
		t.Fatalf("Wrong parent path: %v", parent.fullPath)
	}

	parent = parent.ToParent()
	if parent.fullPath != "a" {
		t.Fatalf("Wrong parent path: %v", parent.fullPath)
	}

	parent = parent.ToParent()
	if parent.fullPath != "" {
		t.Fatalf("Wrong parent path: %v", parent.fullPath)
	}
}

func TestScopedName_ToScopeComplex(t *testing.T) {
	// a.B in scope c.d should be c.d.a.B
	name := ScopedName{
		name:     "B",
		parent:   []string{"a"},
		fullPath: "a.B",
	}

	scope := ScopedName{
		name:     "d",
		parent:   []string{"c"},
		fullPath: "c.d",
	}

	scoped := name.ToScope(scope)
	if scoped.fullPath != "c.d.a.B" {
		t.Fatalf("Wrong scoped path: %v", scoped.fullPath)
	}
	if scoped.name != "B" {
		t.Fatalf("Wrong scoped name: %v", scoped.name)
	}

	if scoped.parent[0] != "c" {
		t.Fatalf("Wrong scoped parent: %v", scoped.parent)
	}
	if scoped.parent[1] != "d" {
		t.Fatalf("Wrong scoped parent: %v", scoped.parent)
	}
	if scoped.parent[2] != "a" {
		t.Fatalf("Wrong scoped parent: %v", scoped.parent)
	}
}

func TestScopedName_ToScopeSimple(t *testing.T) {
	// B in scope d should be c.B
	name := ScopedName{
		name:     "B",
		fullPath: "B",
	}

	scope := ScopedName{
		name:     "d",
		fullPath: "d",
	}

	scoped := name.ToScope(scope)
	if scoped.fullPath != "d.B" {
		t.Fatalf("Wrong scoped path: %v", scoped.fullPath)
	}
	if scoped.name != "B" {
		t.Fatalf("Wrong scoped name: %v", scoped.name)
	}
	if scoped.parent[0] != "d" {
		t.Fatalf("Wrong scoped parent: %v", scoped.parent)
	}
}
