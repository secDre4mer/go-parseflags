package parseflags

import (
	"fmt"
)

func ExampleCreateFlagset() {
	testConfig := struct {
		A int64     `flag:"alpha" description:"an int value"`
		B []string  `flag:"beta" description:"some strings"`
		C bool      `flag:"gamma" hidden:"true"`
	}{
		B: []string{"defaultvalue"},
	}

	args := []string{"--alpha", "10", "--beta", "b", "--gamma"}
	flags := CreateFlagset(&testConfig)
	flags.Parse(args)
	fmt.Println(testConfig.A)
	fmt.Println(testConfig.B)
	fmt.Println(testConfig.C)

	//Output:
	//10
	//[b]
	//true
}
