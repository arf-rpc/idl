package idl

import (
	"github.com/arf-rpc/idl/ast"
)

/*
	Validation must happen in two steps. The first one checks for simple things
	like duplicate structures, enums and services, duplicate fields, methods,
	parameter names, and registers import aliases so it can be resolved later.
	The second step checks for all types by resolving them against the tree
	built from the first step.
*/

type posSet map[string]*ast.Position
