package main

import (
	"os"

	"github.com/Dolyyyy/huma_golang_api_template/internal/templatectl"
)

func main() {
	os.Exit(templatectl.Run(os.Args[1:], os.Stdout, os.Stderr))
}
