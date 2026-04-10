package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/NdumLab/noso/internal/cli"
)

func main() {
	exitCode, err := cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
	}
	os.Exit(exitCode)
}
