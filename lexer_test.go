package idl

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestScanner(t *testing.T) {
	data, err := os.ReadFile("fixtures/contacts.arf")
	require.NoError(t, err)
	scan, errs := lexFile(data, nil)
	require.Empty(t, errs)
	for _, v := range scan {
		fmt.Printf("%s\n", v)
	}
}
