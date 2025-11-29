package idl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	data, err := os.ReadFile("fixtures/full.arf")
	require.NoError(t, err)

	file, errs := lexFile(data, nil)
	require.Empty(t, errs)
	require.NotNil(t, file)
}
