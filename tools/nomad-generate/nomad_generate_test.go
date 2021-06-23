package main

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func setupEnv() {
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
}
func TestGenerate(t *testing.T) {
    setupEnv()
	main()
}

func TestFieldResolveType(t *testing.T) {
	setupEnv()
	main()

	var jobTarget *TargetType

	for _, target := range g.Targets {
		if target.Name == "Job" {
			jobTarget = target
			break
		}
	}

	if jobTarget == nil {
		log.Fatal("unable to locate job target")
	}

	for _, field := range jobTarget.Fields {
		t.Log(fmt.Sprintf("%s: %s", field.Name, field.TypeName))
	}

	fmt.Println("TestFieldResolveType complete")
}
