package fail

import (
	"fmt"
	"testing"
)

func TestPass(t *testing.T) {
	fmt.Printf("Hi, I'm the failing test\n")
	t.FailNow()
}
