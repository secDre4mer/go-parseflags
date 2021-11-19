package parseflags

import (
	"fmt"
	"strconv"
)

func ExampleCreateFlagset() {
	testConfig := struct {
		A int64    `flag:"alpha" description:"an int value"`
		B []string `flag:"beta" description:"some strings"`
		C bool     `flag:"gamma" hidden:"true"`
	}{
		B: []string{"defaultvalue"},
	}

	args := []string{"--alpha", "10", "--beta", "b", "--beta", "b2", "--gamma"}
	flags := CreateFlagset(&testConfig)
	flags.Parse(args)
	fmt.Println(testConfig.A)
	fmt.Println(testConfig.B)
	fmt.Println(testConfig.C)

	//Output:
	//10
	//[b b2]
	//true
}

type Convertible int

func (c *Convertible) Set(val string) error {
	ival, err := strconv.Atoi(val)
	if err == nil {
		*c = Convertible(ival)
	}
	return err
}

func ExampleConvertible() {
	testConfig := struct {
		A Convertible   `flag:"alpha" description:"an int value with custom converter"`
		B []Convertible `flag:"beta" description:"a slice of an int value with custom converter"`
	}{
		A: Convertible(1),
	}

	args := []string{"--alpha", "10", "--beta", "5"}
	flags := CreateFlagset(&testConfig)
	flags.Parse(args)
	fmt.Println(testConfig.A)
	fmt.Println(testConfig.B)

	//Output:
	//10
	//[5]
}

type CustomConvertible int

func ExampleConverter() {
	testConfig := struct {
		A CustomConvertible   `flag:"alpha" description:"an int value with custom converter"`
		B []CustomConvertible `flag:"beta" description:"a slice of an int value with custom converter"`
	}{
		A: CustomConvertible(1),
	}

	RegisterConverter(CustomConvertible(0), func(val string) (interface{}, error) {
		ival, err := strconv.Atoi(val)
		if err == nil {
			return CustomConvertible(ival), nil
		}
		return nil, err
	})

	args := []string{"--alpha", "10", "--beta", "5"}
	flags := CreateFlagset(&testConfig)
	flags.Parse(args)
	fmt.Println(testConfig.A)
	fmt.Println(testConfig.B)

	//Output:
	//10
	//[5]
}

func ExampleRecurseFlagset() {
	testConfig := struct {
		A                  int64 `flag:"alpha" description:"an int value"`
		SubComponentConfig struct {
			B []string `flag:"beta" description:"some strings"`
		} `recurse:"true"`
	}{}

	args := []string{"--alpha", "10", "--beta", "b"}
	flags := CreateFlagset(&testConfig)
	flags.Parse(args)
	fmt.Println(testConfig.A)
	fmt.Println(testConfig.SubComponentConfig.B)

	//Output:
	//10
	//[b]
}
