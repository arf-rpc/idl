package idl

import (
	"fmt"
	"os"
	"testing"

	"github.com/arf-rpc/idl/ast"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	data, err := os.ReadFile("fixtures/full.arf")
	require.NoError(t, err)
	scan, errs := lexFile(data, nil)
	require.Empty(t, errs)
	f, errs := parse("", scan, nil)
	for _, v := range errs {
		fmt.Println(v.Error())
	}
	require.Empty(t, errs)
	fmt.Println()
	ast.Print(f)
}
