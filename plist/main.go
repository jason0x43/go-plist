/*

The plist command parses a plist file and indicates whether the parse was
successful.

*/
package main

import (
	"os"

	"github.com/jason0x43/go-plist"
)

func main() {
	if len(os.Args) != 2 {
		println("usage:", os.Args[0], "PLIST")
		os.Exit(1)
	}

	pl, err := plist.UnmarshalFile(os.Args[1])
	if err != nil {
		println("error unmarshalling plist:", err.Error())
		os.Exit(1)
	}

	println("Read plist of version", pl.Version)

	switch pl.Root.(type) {
	case plist.Dict:
		println("Root is a dict")
	case plist.Array:
		println("Root is an array")
	}

	if err != nil {
		println("error:", err.Error())
	} else {
		println("Parsed plist")
	}
}
