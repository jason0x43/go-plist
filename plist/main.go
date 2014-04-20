package main

import (
	"os"
	"fmt"
	"github.com/jason0x43/go-log"
	"github.com/jason0x43/go-plist"
)

func main() {
	log.Level = log.LEVEL_WARN

	if len(os.Args) != 2 {
		fmt.Printf("usage: %s PLIST\n", os.Args[0])
		os.Exit(1)
	}

	pl, err := plist.UnmarshalFile(os.Args[1])
	if err != nil {
		fmt.Printf("error unmarshalling plist: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Read plist of version %s\n", pl.Version)

	switch pl.Root.(type) {
	case plist.Dict:
		fmt.Printf("Root is a dict\n")
	case plist.Array:
		fmt.Printf("Root is an array\n")
	}

	if err != nil {
		fmt.Println(err)
	} else {
		// fmt.Printf("loaded plist: %v\n", arr[0])
	}
}
