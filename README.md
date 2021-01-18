## go-parseflags in a nutshell

go-parseflags is a Golang library that aims to simplify parsing simple flags. 
Instead of defining the flags in method calls like `StringSliceVarP`,  define them inline using
tags:
```go
type Flags struct {
	MyFlag int64            `flag:"my-flag" description:"my custom flag"`
	MyMultipleFlag []string `flag:"other-flag" description:"my flag that can be specified multiple times"`
	// Add more flags for your program
}

var flags = Flags{
	MyFlag: 2, // default value
}

func init() {
	var flagset = parseflags.CreateFlagset(&flags)
	flagset.Parse(os.Args)
}
```

## Longer explanation

As shown in the example above, you set the properties for your flags using tags on your fields. 
Currently supported tags are:
- `flag` for the flag name (this is customizable, see `FlagsetBuilder.NameTags`)
- `description` for the flag description
- `shorthand` for a single character flag shorthand
- `hidden` (if the tag exists, the flag is not shown in the help)
- `deprecated` (content is a deprecation text that is printed when the flag is used)

By default, most of the primitive types (`int*`, `uint*`, `float*`, `bool`, `string`) are supported. 
You can add support for custom field types by implementing `StringParsable` in your type 
or by calling `RegisterConverter` with your type and a matching conversion method. 

go-parseflag supports slices if the underlying type is supported via `StringParsable` or `RegisterConverter`. 