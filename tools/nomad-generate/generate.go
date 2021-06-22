package nomad_generate

import "fmt"

type StringSliceFlag []string

func (s *StringSliceFlag) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
