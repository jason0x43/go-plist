package plist

type Plist struct {
	Version string
	Root interface{}
}

type Dict map[string]interface{}
type Array []interface{}
