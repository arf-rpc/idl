package idl

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestParser(t *testing.T) {
	file, err := os.Open("fixtures/contacts.arf")
	require.NoError(t, err)
	defer func(file *os.File) { _ = file.Close() }(file)

	fileSet, err := Parse("fixtures/contacts.arf", false)
	require.NoError(t, err)

	fmt.Printf("%s\n", fileSet)

	//assert.Equal(t, []string{"RandomBytesRequest", "RandomBytesResponse", "MessageUsingExternalType"}, tree.DeclaredMessages)
	//assert.Equal(t, []string{"RandomBytesService", "ServiceUsingExternalTypes"}, tree.DeclaredServices)
	//assert.Equal(t, []string{"foo", "bar"}, tree.ImportedFiles)
	//
	//msg, ok := tree.MessageByName("RandomBytesRequest")
	//assert.True(t, ok)
	//assert.Equal(t, "RandomBytesRequest", msg.Name)
	//assert.NotEmpty(t, msg.Comments)
}
