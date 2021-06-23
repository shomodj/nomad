package main

import (
	"os"
	"testing"
)

func TestGenerateEquals(t *testing.T) {

    os.Setenv("GOFILE", "structs.go")
	os.Args = []string{
		"generate",
		"-type=Job",
		"-exclude=Job.Stop",
		"-exclude=Job.CreateIndex",
		"-method=Job.all",
		"-method=Job.Equals",
		"-method=Job.Copy",
		"-method=Job.Diff",
		"-method=Job.Merge",
	}

	main()

}
