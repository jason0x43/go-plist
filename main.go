// Package plist provides an API for reading Apple property lists.
package plist

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// Plist represents a property list.
type Plist struct {
	Version string
	Root    interface{}
}

// Dict represents a property list "dict" element.
type Dict map[string]interface{}

// Array represents a property list "array" element.
type Array []interface{}

// A Decoder represents a plist parser reading from a stream.
type Decoder struct {
	xd *xml.Decoder
}

// NewDecoder creates a Decoder for a given Reader.
func NewDecoder(r io.Reader) *Decoder {
	d := new(Decoder)
	d.xd = xml.NewDecoder(r)
	return d
}

// Unmarshal parses a Plist out of a given array of bytes.
func Unmarshal(data []byte, v *Plist) error {
	dec := NewDecoder(bytes.NewBuffer(data))
	return dec.Decode(v)
}

// UnmarshalFile loads a file and parses a Plist from the loaded data.
// If the file is a binary plist, the plutil system command is used to convert
// it to XML text.
func UnmarshalFile(filename string) (*Plist, error) {
	xmlFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("plist: error opening plist: %s", err)
	}
	defer xmlFile.Close()

	xmlData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, fmt.Errorf("plist: error reading plist file: %s", err)
	}

	if !bytes.HasPrefix(xmlData, []byte("<?xml ")) {
		debug("non-text XML -- assuming binary")
		xmlData, err = exec.Command("plutil", "-convert", "xml1", "-o", "-",
			filename).Output()
		if err != nil || !bytes.HasPrefix(xmlData, []byte("<?xml ")) {
			return nil, fmt.Errorf("plist: invalid plist file " + filename)
		}
	}

	var plist Plist
	err = Unmarshal(xmlData, &plist)
	if err != nil {
		return nil, err
	}

	return &plist, err
}

// Decode executes the Decoder to fill in a Plist structure.
func (d *Decoder) Decode(v *Plist) error {
	start, err := d.decodeStartElement("plist")
	if err != nil {
		return err
	}

	if len(start.Attr) != 1 || start.Attr[0].Name.Local != "version" {
		return fmt.Errorf("plist: missing version")
	}
	v.Version = start.Attr[0].Value

	// Start element of plist content
	se, ee, err := d.decodeStartOrEndElement("", *start)
	if err != nil {
		return err
	}
	if ee != nil {
		// empty plist
		debug("empty plist")
		return nil
	}

	switch se.Name.Local {
	case "dict":
		d, err := d.decodeDict(*se)
		if err != nil {
			return err
		}
		debug("read dict: %s", d)
		v.Root = d
	case "array":
		a, err := d.decodeArray(*se)
		if err != nil {
			return err
		}
		debug("read array: %s", a)
		v.Root = a
	default:
		return fmt.Errorf("plist: bad root element: must be dict or array")
	}

	return err
}

func (d *Decoder) nextElement() (xml.Token, error) {
	for {
		t, err := d.xd.Token()
		if err != nil {
			return nil, err
		}

		switch t.(type) {
		case xml.StartElement, xml.EndElement:
			return t, nil
		}
	}
	return nil, nil
}

func (d *Decoder) decodeValue(se *xml.StartElement) (val interface{}, err error) {
	switch se.Name.Local {
	case "dict":
		val, err = d.decodeDict(*se)
	case "array":
		val, err = d.decodeArray(*se)
	case "true":
		val = true
		_, err = d.nextElement()
	case "false":
		val = false
		_, err = d.nextElement()
	case "date":
		val, err = d.decodeDate(*se)
	case "data":
		val, err = d.decodeData(*se)
	case "string":
		val, err = d.decodeString(*se)
	case "real":
		val, err = d.decodeReal(*se)
	case "integer":
		val, err = d.decodeInteger(*se)
	}

	return val, err
}

func (d *Decoder) decodeDict(start xml.StartElement) (Dict, error) {
	trace("reading dict")
	dictMap := map[string]interface{}{}

	// <key>
	se, end, err := d.decodeStartOrEndElement("key", start)
	if err != nil {
		return nil, err
	}
	if end != nil {
		// empty dict
		return nil, nil
	}

	for {
		// read key name
		keyName, err := d.decodeString(*se)
		if err != nil {
			return nil, err
		}

		// read start element
		se, err := d.decodeStartElement("")
		if err != nil {
			return nil, err
		}

		// decode the element value
		val, err := d.decodeValue(se)
		if err != nil {
			return nil, err
		}
		dictMap[keyName] = val

		// get the next key
		se, end, err = d.decodeStartOrEndElement("key", start)
		if err != nil {
			return nil, err
		}
		if end != nil {
			// end of list
			break
		}
	}

	trace("filled in dictMap: %s", dictMap)
	return dictMap, nil
}

func (d *Decoder) decodeArray(start xml.StartElement) (Array, error) {
	trace("reading array")

	var slice []interface{}

	se, ee, err := d.decodeStartOrEndElement("", start)
	if err != nil {
		return nil, err
	}
	if ee != nil {
		// empty array
		return nil, nil
	}

	for {
		// decode the current value
		val, err := d.decodeValue(se)
		if err != nil {
			return nil, err
		}
		slice = append(slice, val)

		// get the next value
		se, ee, err = d.decodeStartOrEndElement("", start)
		if err != nil {
			return nil, err
		}
		if ee != nil {
			// end of array
			break
		}
	}

	return slice, nil
}

func (d *Decoder) decodeAny(start xml.StartElement) (xml.Token, error) {
	t, err := d.xd.Token()
	if err != nil {
		return nil, fmt.Errorf("plist: error reading token: %s", err)
	}

	end, ok := t.(xml.EndElement)
	if ok {
		if end.Name.Local != start.Name.Local {
			return nil, fmt.Errorf("plist: unexpected end tag: %s", end.Name.Local)
		}
		// empty
		return nil, nil
	}

	tok := xml.CopyToken(t)

	next, err := d.nextElement()
	if err != nil {
		return nil, fmt.Errorf("plist: error reading token: %s", err)
	}
	end, ok = next.(xml.EndElement)
	if !ok || end.Name.Local != start.Name.Local {
		// empty
		return nil, fmt.Errorf("plist: unexpected end tag: %s", end.Name.Local)
	}

	return tok, nil
}

func (d *Decoder) decodeString(start xml.StartElement) (string, error) {
	t, err := d.decodeAny(start)
	if err != nil {
		return "", err
	}
	if t == nil {
		trace("read empty string")
		return "", nil
	}

	cd, ok := t.(xml.CharData)
	if !ok {
		return "", fmt.Errorf("plist: expected character data")
	}

	trace("read string '%s'", string(cd))

	return string(cd), nil
}

func (d *Decoder) decodeNonEmptyString(start xml.StartElement) (string, error) {
	str, err := d.decodeString(start)
	if err != nil {
		return str, err
	}
	if len(str) == 0 {
		return str, fmt.Errorf("plist: expected non-empty string")
	}
	return str, nil
}

func (d *Decoder) decodeInteger(start xml.StartElement) (int64, error) {
	str, err := d.decodeNonEmptyString(start)
	if err != nil {
		return 0, err
	}

	trace("read integer: %s", str)

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("plist: invalid integer value '%s'", str)
	}

	return val, nil
}

func (d *Decoder) decodeReal(start xml.StartElement) (float64, error) {
	str, err := d.decodeNonEmptyString(start)
	if err != nil {
		return 0, err
	}

	trace("read real: %s", str)

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("plist: invalid float value '%s'", str)
	}

	return val, nil
}

func (d *Decoder) decodeDate(start xml.StartElement) (time.Time, error) {
	str, err := d.decodeNonEmptyString(start)
	if err != nil {
		return time.Time{}, err
	}

	trace("read date: %s", str)

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}, fmt.Errorf("plist: invalid date value '%s'", str)
	}

	return t, nil
}

func (d *Decoder) decodeData(start xml.StartElement) ([]byte, error) {
	trace("reading data")
	str, err := d.decodeString(start)
	if err != nil {
		return nil, err
	}
	if str == "" {
		return nil, nil
	}

	trace("read data: %s", str)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, fmt.Errorf("plist: error decoding data '%s'", str)
	}

	return data, nil
}

func (d *Decoder) decodeStartElement(expected string) (*xml.StartElement, error) {
	trace("reading start element '%s'", expected)
	t, err := d.nextElement()
	if err != nil {
		return nil, err
	}
	se, ok := t.(xml.StartElement)
	if !ok {
		return nil, fmt.Errorf("plist: expected StartElement, saw %T", t)
	}
	if expected != "" && se.Name.Local != expected {
		return nil, fmt.Errorf("plist: unexpected key name '%s'", se.Name.Local)
	}
	return &se, nil
}

func (d *Decoder) decodeEndElement(expected string) (*xml.EndElement, error) {
	trace("reading end element '%s'", expected)
	t, err := d.nextElement()
	if err != nil {
		return nil, err
	}
	ee, ok := t.(xml.EndElement)
	if !ok {
		return nil, fmt.Errorf("plist: expected EndElement")
	}
	if expected != "" && ee.Name.Local != expected {
		return nil, fmt.Errorf("bad key name")
	}
	trace("  read element '%s'", ee)
	return &ee, nil
}

func (d *Decoder) decodeStartOrEndElement(expected string, start xml.StartElement) (
	*xml.StartElement, *xml.EndElement, error) {
	trace("reading start element '%s' or end element", expected)
	t, err := d.nextElement()
	if err != nil {
		return nil, nil, err
	}
	se, ok := t.(xml.StartElement)
	if !ok {
		ee, ok := t.(xml.EndElement)
		if ok {
			if ee.Name.Local == start.Name.Local {
				trace("  read end element '%s'", ee)
				return nil, &ee, nil
			}
			return nil, nil, fmt.Errorf("plist: unexpected end element '%s'", ee.Name.Local)
		}
		return nil, nil, fmt.Errorf("plist: expected StartElement, saw %T", se)
	}
	if expected != "" && se.Name.Local != expected {
		return nil, nil, fmt.Errorf("plist: unexpected key name '%s'", se.Name.Local)
	}
	trace("  read start element '%s'", se)
	return &se, nil, nil
}

// support /////////////////////////////////////////////////////////////

func debug(format string, args ...interface{}) {
	// log.Printf(format, args...)
}

func trace(format string, args ...interface{}) {
	// log.Printf(format, args...)
}
