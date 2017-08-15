package main

import (
	"fmt"
	"os"

	"github.com/munnerz/keepalived-cloud-provider/keepalivedcp"
	"github.com/spf13/pflag"
)

var (
	version     string = "unknwon"
	versionFlag bool
)

func addVersionFlag() {
	pflag.BoolVar(&versionFlag, "version", false, "Print version information and quit")
}

func printAndExitIfRequested() {
	if versionFlag {
		fmt.Printf("%s %s\n", keepalivedcp.ProviderName, version)
		os.Exit(0)
	}
}
