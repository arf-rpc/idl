package idl

import (
	"fmt"
	"testing"
)

func TestIDL(t *testing.T) {
	files, err := ParseFile("./fixtures/contacts.arf", nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(files)
}
