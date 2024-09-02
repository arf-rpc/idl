package idl

import (
	"github.com/davecgh/go-spew/spew"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	file, err := os.Open("fixtures/contacts.arf")
	require.NoError(t, err)
	defer func(file *os.File) { _ = file.Close() }(file)

	fileSet, err := Parse("fixtures/contacts.arf", false)
	require.NoError(t, err)
	require.Len(t, fileSet, 1)

	f := fileSet[0]
	tree := f.Tree
	require.NotNil(t, tree)
	spew.Dump(tree)

	assert.Empty(t, tree.Imports)
	assert.Equal(t, "org.example.contacts", tree.Package.Name)
	assert.Empty(t, tree.Enums)

	s0 := tree.Structures[0]
	assert.Equal(t, s0.Name, "Contact")
	assert.Len(t, s0.Fields, 8)
	spew.Dump(s0)
}
