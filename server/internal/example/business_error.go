package example

import "errors"

// ErrReservedName is returned when a caller tries to create an example
// using a name this module reserves for internal use.
var ErrReservedName = errors.New("name is reserved")
