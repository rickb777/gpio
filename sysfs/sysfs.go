// Package sysfs provides functions to read/write special files in sysfs, such
// as "/sys/bus/iio/devices/iio:device0".
// This can be used for IIO, GPIO etc.
package sysfs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//-------------------------------------------------------------------------------------------------

// Path is an extensible string representing a file or directory path. In this package, a
// root instance should be used to hold the base of the sysfs nodes of interest so that the
// nodes can be directly accessed from it via [Path.Join].
type Path string

func (path Path) Join(extra string) Path {
	return Path(filepath.Join(string(path), extra))
}

func (path Path) String() string {
	return string(path)
}

func (path Path) IsExistingFile() (bool, error) {
	return existsWithPredicate(path, func(info os.FileInfo) bool {
		return info.Mode().IsRegular()
	})
}

func (path Path) IsExistingDirectory() (bool, error) {
	return existsWithPredicate(path, func(info os.FileInfo) bool {
		return info.Mode().IsDir()
	})
}

func existsWithPredicate(path Path, predicate func(os.FileInfo) bool) (bool, error) {
	info, err := os.Stat(string(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return predicate(info), nil
}

//-------------------------------------------------------------------------------------------------

// PathOrString is a type interface allowing a parameter choice for the GetXxx and SetXxx methods.
type PathOrString interface{ string | Path }

//-------------------------------------------------------------------------------------------------

// GetString reads a setting from a sysfs node.
func GetString[P PathOrString](node P) (string, error) {
	return readTextFile(string(node))
}

// GetValue gets an arbitrary value of any type given a parsing function for
// that type.
func GetValue[T any, P PathOrString](node P, parse func(string) (T, error)) (T, error) {
	s, err := GetString(string(node))
	if err != nil {
		var zero T
		return zero, err
	}
	return parse(s)
}

// GetInt64 gets an integer as a signed int64.
func GetInt64[P PathOrString](node P) (int64, error) {
	return GetValue(node, parseInt64)
}

// GetHex64 gets a hexadecimal integer, discarding any "0x" prefix if present.
func GetHex64[P PathOrString](node P) (int64, error) {
	return GetValue(node, parseHex64)
}

// GetInt gets an integer as an int.
func GetInt[P PathOrString](node P) (int, error) {
	return GetValue(node, strconv.Atoi)
}

// GetBool gets a boolean value. True values are 1, t, or true.
// False values are 0, f, or false. The case is ignored.
func GetBool[P PathOrString](node P) (bool, error) {
	return GetValue(node, strconv.ParseBool)
}

// GetStrings gets a slice of space-separated strings. If there is a
// surrounding "[" and "]", these are discarded.
func GetStrings[P PathOrString](node P) ([]string, error) {
	s, err := GetString(string(node))
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
	}
	return strings.Split(s, " "), nil
}

//-------------------------------------------------------------------------------------------------

func parseInt64(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

func parseHex64(s string) (int64, error) {
	if strings.HasPrefix(s, "0x") {
		s = s[2:]
	}
	return strconv.ParseInt(s, 16, 64)
}

//-------------------------------------------------------------------------------------------------

// SetString writes a setting to a sysfs node.
func SetString[P PathOrString](node P, value string) error {
	return writeTextFile(string(node), value)
}

// SetInt64 writes an integer in base 10.
func SetInt64[P PathOrString](node P, value int64) error {
	return SetString(string(node), strconv.FormatInt(value, 10))
}

// SetHex64 writes an integer in base 16 prefixed by [HexPrefix].
func SetHex64[P PathOrString](node P, value int64) error {
	return SetString(string(node), HexPrefix+strconv.FormatInt(value, 16))
}

// SetInt writes an integer in base 10.
func SetInt[P PathOrString](node P, value int) error {
	return SetString(string(node), strconv.Itoa(value))
}

// SetBool writes a boolean value as 1 or 0.
func SetBool[P PathOrString](node P, value bool) error {
	if value {
		return SetString(string(node), "1")
	}
	return SetString(string(node), "0")
}

//-------------------------------------------------------------------------------------------------
// Seams for configuration & testing

var readTextFile = func(file string) (string, error) {
	Debugf("Reading %s\n", file)
	bs, err := os.ReadFile(file)
	return strings.TrimSpace(string(bs)), err
}

var writeTextFile = func(file, contents string) error {
	Debugf("Writing %s > %s\n", contents, file)
	return os.WriteFile(file, []byte(contents), 0666)
}

var HexPrefix = "0x"

//-------------------------------------------------------------------------------------------------

// Debugf can be assigned to [log.Printf] to enable diagnostics.
var Debugf = func(format string, a ...any) {}
