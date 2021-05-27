package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/carbonin/iso-stream/injector"
)

func main() {
	inPath := flag.String("in", "isos/rhcos-4.6.1-x86_64-live.x86_64.iso", "input ISO path")
	outPath := flag.String("out", "isos/my-rhcos.iso", "output ISO path")
	ignitionContent := flag.String("ignition", "some-ignition-content-here", "ignition content to add to the iso")
	flag.Parse()

	baseISO, err := os.Open(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening base iso file %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	defer baseISO.Close()

	outputISO, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening output iso file %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	defer outputISO.Close()

	r, err := injector.NewRHCOSStreamReader(baseISO, *ignitionContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create rhcos stream editor: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Reading data from stream editor to file ...\n")
	fmt.Printf("Ignition: %s\n", *ignitionContent)

	n, err := io.Copy(outputISO, r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy from stream editor: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Copied %d bytes from stream editor to %s\n", n, *outPath)
}
