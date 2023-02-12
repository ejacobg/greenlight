package data

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// A valid runtime value looks like this: "<runtime> mins"
// The quotes are part of a valid value.
var ErrInvalidRuntimeFormat = errors.New("invalid runtime format")

type Runtime int32

func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}

// UnmarshalJSON expects the incoming value to be of the form: "<runtime> mins"
func (r *Runtime) UnmarshalJSON(jsonValue []byte) error {
	// Remove surrounding quotes from the value.
	unquotedJSONValue, err := strconv.Unquote(string(jsonValue))
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	parts := strings.Split(unquotedJSONValue, " ")

	if len(parts) != 2 || parts[1] != "mins" {
		return ErrInvalidRuntimeFormat
	}

	// The <runtime> value should be a valid int32.
	i, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return ErrInvalidRuntimeFormat
	}

	// Apply the decoded value to the Runtime object.
	*r = Runtime(i)
	return nil
}
