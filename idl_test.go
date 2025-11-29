package idl

import (
	"testing"

	"github.com/arf-rpc/idl/ast"
	"github.com/stretchr/testify/require"
)

func TestFullParse(t *testing.T) {
	fe, err := New("fixtures/full.arf")
	require.NoError(t, err)
	require.NotNil(t, fe)
	err = fe.Run()
	require.NoError(t, err)
}

func TestMethodParamsMustBeStructOrEnum(t *testing.T) {
	cases := []string{
		"package p; struct S{} service X{ M(i int32); }",             // primitive param
		"package p; struct S{} service X{ M() -> int32; }",           // primitive return
		"package p; struct S{} service X{ M(i optional<int32>); }",   // optional param
		"package p; struct S{} service X{ M(i array<int32>); }",      // array param
		"package p; struct S{} service X{ M(i map<string,int32>); }", // map param
		"package p; enum E{A=1;} service X{ M(i optional<E>); }",     // optional enum param (non-struct)
	}
	for _, src := range cases {
		tokens, errs := lexFile([]byte(src), nil)
		require.Empty(t, errs, src)
		fe, errs := parse("", tokens, nil)
		require.Empty(t, errs, src)
		require.Error(t, validatePhase2(map[string]*ast.File{"": fe}, ""), src)
	}
}

func TestStreamMustBeLast(t *testing.T) {
	src := `package p; struct S{} service X{ M(stream S, x S); M(x S, stream S, y S); M() -> (stream S, S); }`
	tokens, errs := lexFile([]byte(src), nil)
	require.Empty(t, errs)
	_, errs = parse("example.arf", tokens, nil)
	require.NotEmpty(t, errs)
}

func TestMapKeyTypes(t *testing.T) {
	good := []string{
		"package p; struct K{} struct V{} struct S{ m map<string,V>; }",
		"package p; enum E{A=1;} struct V{} struct S{ m map<E,V>; }",
		"package p; struct K{} struct V{} struct S{ m map<K,V>; }",
	}
	bad := []string{
		"package p; struct V{} struct S{ m map<optional<string>,V>; }",
		"package p; struct V{} struct S{ m map<array<string>,V>; }",
		"package p; struct V{} struct S{ m map<map<string,string>,V>; }",
	}
	for _, src := range good {
		tokens, errs := lexFile([]byte(src), nil)
		require.Empty(t, errs, src)
		fe, errs := parse("", tokens, nil)
		require.Empty(t, errs, src)
		require.NoError(t, validatePhase2(map[string]*ast.File{"": fe}, ""), src)
	}
	for _, src := range bad {
		tokens, errs := lexFile([]byte(src), nil)
		require.Empty(t, errs, src)
		fe, errs := parse("", tokens, nil)
		require.Empty(t, errs, src)
		require.Error(t, validatePhase2(map[string]*ast.File{"": fe}, ""), src)
	}
}

func TestPackageAndImportCasing(t *testing.T) {
	bad := []string{
		`package Bad.Case; struct S{ f string; }`,
		`package good.case; import "other.arf" as BadAlias; struct S{ f string; }`,
	}
	for _, src := range bad {
		tokens, errs := lexFile([]byte(src), nil)
		require.Empty(t, errs, src)
		_, errs = parse("", tokens, nil)
		require.NotEmpty(t, errs, src)
	}
}

func TestServiceReopenDivergence(t *testing.T) {
	src := `package p; struct S{} service X{ M(i S); } service X{ M(i S, stream S); }`
	tokens, errs := lexFile([]byte(src), nil)
	require.Empty(t, errs)
	fe, errs := parse("", tokens, nil)
	require.Empty(t, errs)
	files := map[string]*ast.File{"": fe}
	require.NoError(t, validatePhase1(files, ""))
	require.NoError(t, validatePhase2(files, ""))
	require.Error(t, validatePhase3(files, ""))
}

func TestUnresolvedTypes(t *testing.T) {
	src := `package p; struct S{ f Missing; }`
	tokens, errs := lexFile([]byte(src), nil)
	require.Empty(t, errs)
	fe, errs := parse("", tokens, nil)
	require.Empty(t, errs)
	err := validatePhase2(map[string]*ast.File{"": fe}, "")
	require.Error(t, err)
}

func TestStreamPlacementAndUniqueness(t *testing.T) {
	bad := []string{
		"package p; struct S{} service X{ M(stream S, x S); }",            // stream not last in params
		"package p; struct S{} service X{ M(x S, stream S, y S); }",       // stream not last in params
		"package p; struct S{} service X{ M() -> (stream S, S); }",        // stream not last in returns
		"package p; struct S{} service X{ M(stream S, stream S); }",       // two input streams
		"package p; struct S{} service X{ M() -> (stream S, stream S); }", // two output streams
	}
	for _, src := range bad {
		tokens, errs := lexFile([]byte(src), nil)
		require.Empty(t, errs, src)
		_, errs = parse("example.arf", tokens, nil)
		require.NotEmpty(t, errs)
	}
}

func TestAnnotationParamsMustBeStrings(t *testing.T) {
	src := `@ann(123) package p; struct S{ f string; }`
	tokens, errs := lexFile([]byte(src), nil)
	require.Empty(t, errs)
	_, errs = parse("", tokens, nil)
	require.NotEmpty(t, errs)
}

func TestServiceReopenIdenticalPasses(t *testing.T) {
	src := `package p; struct S{} service X{ M(i S); } service X{ M(i S); }`
	tokens, errs := lexFile([]byte(src), nil)
	require.Empty(t, errs)
	fe, errs := parse("", tokens, nil)
	require.Empty(t, errs)
	files := map[string]*ast.File{"": fe}
	require.NoError(t, validatePhase1(files, ""))
	require.NoError(t, validatePhase2(files, ""))
	require.NoError(t, validatePhase3(files, ""))
}

func TestDuplicateImportAliases(t *testing.T) {
	fe, err := New("fixtures/duplicate_import_aliases.arf")
	require.NoError(t, err)
	require.NotNil(t, fe)
	err = fe.Run()
	require.Error(t, err)
}

func TestReservedWordsAsIdentifiers(t *testing.T) {
	cases := []string{
		`package p; struct struct{ f string; }`,
		`package p; struct S{ map string; }`,    // field named reserved word
		`package p; service service{ M(i S); }`, // service name reserved
	}
	for _, src := range cases {
		tokens, errs := lexFile([]byte(src), nil)
		require.Empty(t, errs, src)
		_, errs = parse("", tokens, nil)
		require.NotEmpty(t, errs, src)
	}
}
