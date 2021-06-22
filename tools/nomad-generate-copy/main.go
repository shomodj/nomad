package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var excludedFields stringSliceFlag
var typeName string

func main() {
	flag.Var(&excludedFields, "exclude", "list of fields to exclude from Copy")
	flag.StringVar(&typeName, "type", "", "type for which to generate Copy method")
	flag.Parse()

	if typeName == "" {
		fmt.Println("missing name of type to generate Copy")
		os.Exit(2)
	}

	fmt.Printf("generating Copy for %s\n", typeName)
	for _, exclude := range excludedFields {
		fmt.Printf(" excluding field %s\n", exclude)
	}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GO") {
			fmt.Printf(" %s\n", kv)
		}
	}
}
