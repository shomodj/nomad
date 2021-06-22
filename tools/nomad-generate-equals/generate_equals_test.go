package main

import (
	"os"
	"testing"
)

func TestGenerateEquals(t *testing.T) {

	typeNames = append(typeNames, "Job")
	excludedFields = append(excludedFields, "Stop")
	excludedFields = append(excludedFields, "CreateIndex")
    os.Setenv("GOFILE", "structs.go")
	generateEquals()

}
