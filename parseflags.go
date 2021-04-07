package parseflags

import (
	"encoding/csv"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

type ElementFilter interface {
	Filter(field reflect.StructField) bool
}

type FlagsetBuilder struct {
	Filter   ElementFilter
	NameTags []string
}

func NewBuilder() *FlagsetBuilder {
	return &FlagsetBuilder{
		NameTags: []string{"flag"},
	}
}

func (b *FlagsetBuilder) SetFilter(filter ElementFilter) *FlagsetBuilder {
	b.Filter = filter
	return b
}

func (b *FlagsetBuilder) SetNameTags(tags ...string) *FlagsetBuilder {
	b.NameTags = tags
	return b
}

func CreateFlagset(config interface{}) *flag.FlagSet {
	return NewBuilder().Build(config)
}

func (b *FlagsetBuilder) Build(config interface{}) *flag.FlagSet {
	reflectConfig := reflect.ValueOf(config)
	if reflectConfig.Kind() != reflect.Ptr {
		panic("Must pass pointer to struct containing configuration")
	}
	reflectConfig = reflectConfig.Elem()
	configType := reflectConfig.Type()
	var flags = flag.NewFlagSet("", flag.ContinueOnError)
	for i := 0; i < configType.NumField(); i++ {
		structField := configType.Field(i)

		var name string
		for _, tag := range b.NameTags {
			tagValue, tagExists := structField.Tag.Lookup(tag)
			if tagExists && tagValue != "-" {
				name = tagValue
				break
			}
		}
		if name != "" {
			if b.Filter != nil && !b.Filter.Filter(structField) {
				continue
			}

			description := structField.Tag.Get("description")
			shorthand := structField.Tag.Get("shorthand")

			field := reflectConfig.Field(i)

			var variable = makeVar(field.Addr().Interface())
			createdFlag := flags.VarPF(variable, name, shorthand, description)
			if field.Type() == reflect.TypeOf(false) {
				createdFlag.NoOptDefVal = "true"
			}
			_, isHidden := structField.Tag.Lookup("hidden")
			if isHidden {
				createdFlag.Hidden = true
			}
			deprecationText, isDeprecated := structField.Tag.Lookup("deprecated")
			if isDeprecated {
				createdFlag.Deprecated = deprecationText
			}
			aliases, hasAliases := structField.Tag.Lookup("alias")
			if hasAliases {
				for _, alias := range strings.Split(aliases, ",") {
					aliasFlag := flags.VarPF(variable, alias, "", "")
					aliasFlag.Hidden = true
					if field.Type() == reflect.TypeOf(false) {
						aliasFlag.NoOptDefVal = "true"
					}
				}
			}
		}
	}
	return flags
}

type NamedType interface {
	Type() string
}

type StringParsable interface {
	Set(value string) (err error)
}

type generalPurposeVar struct {
	Target    interface{}
	Converter func(string) (interface{}, error)
	changed   bool
}

func (g *generalPurposeVar) isSlice() bool {
	return reflect.TypeOf(g.Target).Elem().Kind() == reflect.Slice
}

func (g *generalPurposeVar) Set(value string) (err error) {
	if g.isSlice() {
		csvReader := csv.NewReader(strings.NewReader(value))
		sliceValues, err := csvReader.Read()
		if err != nil {
			return err
		}
		sliceType := reflect.TypeOf(g.Target).Elem()
		valueSlice := reflect.MakeSlice(sliceType, len(sliceValues), len(sliceValues))
		for i, value := range sliceValues {
			convertedValue, err := g.Converter(value)
			if err != nil {
				return err
			}
			valueSlice.Index(i).Set(reflect.ValueOf(convertedValue).Convert(sliceType.Elem()))
		}
		if !g.changed {
			reflect.Indirect(reflect.ValueOf(g.Target)).Set(valueSlice)
		} else {
			reflect.Indirect(reflect.ValueOf(g.Target)).Set(
				reflect.AppendSlice(reflect.Indirect(reflect.ValueOf(g.Target)), valueSlice))
		}
	} else {
		convertedValue, err := g.Converter(value)
		if err != nil {
			return err
		}
		newTargetValue := reflect.ValueOf(convertedValue).Convert(reflect.TypeOf(g.Target).Elem())
		reflect.Indirect(reflect.ValueOf(g.Target)).Set(newTargetValue)
	}
	g.changed = true
	return
}

func (g *generalPurposeVar) String() string {
	if !g.isSlice() {
		return fmt.Sprintf("%v", reflect.Indirect(reflect.ValueOf(g.Target)).Interface())
	} else {
		value := reflect.Indirect(reflect.ValueOf(g.Target))
		var stringElements = make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			stringElements[i] = fmt.Sprintf("%v", value.Index(i).Interface())
		}
		var csvBuffer strings.Builder
		csvWriter := csv.NewWriter(&csvBuffer)
		csvWriter.Write(stringElements)
		csvWriter.Flush()
		return "[" + strings.TrimSuffix(csvBuffer.String(), "\n") + "]"
	}
}

func (g *generalPurposeVar) Type() string {
	if named, isNamed := g.Target.(NamedType); isNamed {
		return named.Type()
	}
	targetType := reflect.TypeOf(g.Target).Elem()
	var suffix string
	if targetType.Kind() == reflect.Slice {
		suffix = "Slice"
		targetType = targetType.Elem()
	}
	return targetType.Name() + suffix
}

type Converter func(string) (interface{}, error)

var gpVarConverters = map[reflect.Type]Converter{}

func RegisterConverter(targetType interface{}, converter Converter) {
	gpVarConverters[reflect.TypeOf(targetType)] = converter
}

func makeVar(target interface{}) *generalPurposeVar {
	pointerType := reflect.TypeOf(target)
	targetType := pointerType.Elem()
	if targetType.Kind() == reflect.Slice {
		targetType = targetType.Elem()
		pointerType = reflect.PtrTo(targetType)
	}
	var converter Converter
	if pointerType.Implements(reflect.TypeOf((*StringParsable)(nil)).Elem()) {
		converter = func(val string) (interface{}, error) {
			zeroVal := reflect.New(targetType)
			err := zeroVal.Interface().(StringParsable).Set(val)
			return zeroVal.Elem().Interface(), err
		}
	} else if registered, hasRegistered := gpVarConverters[targetType]; hasRegistered {
		converter = registered
	} else {
		panic("No converter available for type " + targetType.Name())
	}
	return &generalPurposeVar{
		Target:    target,
		Converter: converter,
	}
}

func init() {
	RegisterConverter(int64(0), func(val string) (interface{}, error) {
		return strconv.ParseInt(val, 10, 64)
	})
	RegisterConverter(int32(0), func(val string) (interface{}, error) {
		return strconv.ParseInt(val, 10, 32)
	})
	RegisterConverter(int16(0), func(val string) (interface{}, error) {
		return strconv.ParseInt(val, 10, 16)
	})
	RegisterConverter(int8(0), func(val string) (interface{}, error) {
		return strconv.ParseInt(val, 10, 8)
	})
	RegisterConverter(uint64(0), func(val string) (interface{}, error) {
		return strconv.ParseUint(val, 10, 64)
	})
	RegisterConverter(uint32(0), func(val string) (interface{}, error) {
		return strconv.ParseUint(val, 10, 32)
	})
	RegisterConverter(uint16(0), func(val string) (interface{}, error) {
		return strconv.ParseUint(val, 10, 16)
	})
	RegisterConverter(uint8(0), func(val string) (interface{}, error) {
		return strconv.ParseUint(val, 10, 8)
	})
	RegisterConverter(0, func(val string) (interface{}, error) {
		return strconv.ParseInt(val, 10, 64)
	})
	RegisterConverter(uint(0), func(val string) (interface{}, error) {
		return strconv.ParseUint(val, 10, 64)
	})
	RegisterConverter(float64(0), func(val string) (interface{}, error) {
		return strconv.ParseFloat(val, 64)
	})
	RegisterConverter(float32(0), func(val string) (interface{}, error) {
		return strconv.ParseFloat(val, 32)
	})
	RegisterConverter(false, func(val string) (interface{}, error) {
		return strconv.ParseBool(val)
	})
	RegisterConverter("", func(val string) (interface{}, error) {
		return val, nil
	})
}
