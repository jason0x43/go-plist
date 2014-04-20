package plist

import (
	"io"
	"os"
	"fmt"
	"time"
	"bytes"
	"strconv"
	"io/ioutil"
	"encoding/xml"
	"encoding/base64"
	"github.com/jason0x43/go-log"
)

type Decoder struct {
	xd  *xml.Decoder
}

func Unmarshal(data []byte, v *Plist) error {
	dec := NewDecoder(bytes.NewBuffer(data))
	return dec.Decode(v)
}

func UnmarshalFile(filename string) (*Plist, error) {
	xmlFile, err := os.Open("info.plist")
	if err != nil {
		return nil, fmt.Errorf("plist: error opening plist: %s", err)
	}
	defer xmlFile.Close()

	xmlData, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		return nil, fmt.Errorf("plist: error reading plist file: %s", err)
	}

	var plist Plist
	err = Unmarshal(xmlData, &plist)
	if err != nil {
		return nil, err
	}

	return &plist, err
}

func NewDecoder(r io.Reader) *Decoder {
	d := new(Decoder)
	d.xd = xml.NewDecoder(r)
	return d
}

func (d *Decoder) Decode(v *Plist) error {
	return d.parsePlist(v)
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

func (d *Decoder) parsePlist(v *Plist) error {
	// <plist version="xxxx">
	start, err := d.readStartElement("plist")
	if err != nil {
		return err
	}

	if len(start.Attr) != 1 || start.Attr[0].Name.Local != "version" {
		return fmt.Errorf("plist: missing version")
	}
	v.Version = start.Attr[0].Value

	// Start element of plist content
	se, ee, err := d.readStartOrEndElement("", *start)
	if err != nil {
		return err
	}
	if ee != nil {
		// empty plist
		log.Trace("empty plist")
		return nil
	}

	switch se.Name.Local {
	case "dict":
		d, err := d.readDict(*se)
		if err != nil {
			return err
		}
		log.Trace("read dict: %s", d)
		v.Root = d
	case "array":
		a, err := d.readArray(*se)
		if err != nil {
			return err
		}
		log.Trace("read array: %s", a)
		v.Root = a
	default:
		return fmt.Errorf("plist: bad root element: must be dict or array")
	}

	return err
}

func (d *Decoder) readDict(start xml.StartElement) (Dict, error) {
	log.Trace("reading dict")
	dictMap := map[string]interface{}{}

	// <key>
	se, end, err := d.readStartOrEndElement("key", start)
	if err != nil {
		return nil, err
	}
	if end != nil {
		// empty dict
		return nil, nil
	}

	for {
		// read key name
		keyName, err := d.readString(*se)
		if err != nil {
			return nil, err
		}

		// read start element
		se, err := d.readStartElement("")
		if err != nil {
			return nil, err
		}

		switch se.Name.Local {
		case "dict":
			m, err := d.readDict(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = m
		case "array":
			a, err := d.readArray(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = a
		case "true":
			_, err := d.nextElement()
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = true
		case "false":
			_, err := d.nextElement()
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = false
		case "date":
			t, err := d.readDate(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = t
		case "data":
			buf, err := d.readData(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = buf
		case "string":
			s, err := d.readString(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = s
		case "real":
			f, err := d.readReal(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = f
		case "integer":
			i, err :=  d.readInteger(*se)
			if err != nil {
				return nil, err
			}
			dictMap[keyName] = i
		}

		se, end, err = d.readStartOrEndElement("key", start)
		if err != nil {
			return nil, err
		}
		if end != nil {
			// end of list
			break
		}
	}

	log.Trace("filled in dictMap: %s", dictMap)
	return dictMap, nil
}

func (d *Decoder) readArray(start xml.StartElement) (Array, error) {
	log.Trace("reading array")

	var slice []interface{}

	se, ee, err := d.readStartOrEndElement("", start)
	if err != nil {
		return nil, err
	}
	if ee != nil {
		// empty array
		return nil, nil
	}

	for {
		switch se.Name.Local {
		case "dict":
			m, err := d.readDict(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, m)
		case "array":
			a, err := d.readArray(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, a)
		case "true":
			_, err := d.nextElement()
			if err != nil {
				return nil, err
			}
			slice = append(slice, true)
		case "false":
			_, err := d.nextElement()
			if err != nil {
				return nil, err
			}
			slice = append(slice, false)
		case "date":
			t, err := d.readDate(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, t)
		case "data":
			buf, err := d.readData(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, buf)
		case "string":
			s, err := d.readString(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, s)
		case "real":
			f, err := d.readReal(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, f)
		case "integer":
			i, err :=  d.readInteger(*se)
			if err != nil {
				return nil, err
			}
			slice = append(slice, i)
		}

		se, ee, err = d.readStartOrEndElement("", start)
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

func (d *Decoder) readAny(start xml.StartElement) (xml.Token, error) {
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

func (d *Decoder) readString(start xml.StartElement) (string, error) {
	t, err := d.readAny(start)
	if err != nil {
		return "", err
	}
	if t == nil {
		log.Trace("read empty string")
		return "", nil
	}

	cd, ok := t.(xml.CharData)
	if !ok {
		return "", fmt.Errorf("plist: expected character data")
	}

	log.Trace("read string '%s'", string(cd))

	return string(cd), nil
}

func (d *Decoder) readNonEmptyString(start xml.StartElement) (string, error) {
	str, err := d.readString(start)
	if err != nil {
		return str, err
	}
	if len(str) == 0 {
		return str, fmt.Errorf("plist: expected non-empty string")
	}
	return str, nil
}

func (d *Decoder) readInteger(start xml.StartElement) (int64, error) {
	str, err := d.readNonEmptyString(start)
	if err != nil {
		return 0, err
	}

	log.Trace("read integer: %s", str)

	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("plist: invalid integer value '%s'", str)
	}

	return val, nil
}

func (d *Decoder) readReal(start xml.StartElement) (float64, error) {
	str, err := d.readNonEmptyString(start)
	if err != nil {
		return 0, err
	}

	log.Trace("read real: %s", str)

	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, fmt.Errorf("plist: invalid float value '%s'", str)
	}

	return val, nil
}

func (d *Decoder) readDate(start xml.StartElement) (time.Time, error) {
	str, err := d.readNonEmptyString(start)
	if err != nil {
		return time.Time{}, err
	}

	log.Trace("read date: %s", str)

	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return time.Time{}, fmt.Errorf("plist: invalid date value '%s'", str)
	}

	return t, nil
}

func (d *Decoder) readData(start xml.StartElement) ([]byte, error) {
	log.Trace("reading data")
	str, err := d.readString(start)
	if err != nil {
		return nil, err
	}
	if str == "" {
		return nil, nil
	}

	log.Trace("read data: %s", str)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, fmt.Errorf("plist: error decoding data '%s'", str)
	}

	return data, nil
}

func (d *Decoder) readStartElement(expected string) (*xml.StartElement, error) {
	// log.Trace("reading start element '%s'", expected)
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

func (d *Decoder) readEndElement(expected string) (*xml.EndElement, error) {
	log.Trace("reading end element '%s'", expected)
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
	log.Trace("  read element '%s'", ee)
	return &ee, nil
}

func (d *Decoder) readStartOrEndElement(expected string, start xml.StartElement) (
	*xml.StartElement, *xml.EndElement, error) {
	// log.Trace("reading start element '%s' or end element", expected)
	t, err := d.nextElement()
	if err != nil {
		return nil, nil, err
	}
	se, ok := t.(xml.StartElement)
	if !ok {
		ee, ok := t.(xml.EndElement)
		if ok {
			if ee.Name.Local == start.Name.Local {
				// log.Trace("  read end element '%s'", ee)
				return nil, &ee, nil
			}
			return nil, nil, fmt.Errorf("plist: unexpected end element '%s'", ee.Name.Local)
		}
		return nil, nil, fmt.Errorf("plist: expected StartElement, saw %T", se)
	}
	if expected != "" && se.Name.Local != expected {
		return nil, nil, fmt.Errorf("plist: unexpected key name '%s'", se.Name.Local)
	}
	// log.Trace("  read start element '%s'", se)
	return &se, nil, nil
}
