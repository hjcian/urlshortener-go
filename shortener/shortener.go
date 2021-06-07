package shortener

import (
	"crypto/md5"
	"errors"
	"fmt"
	"time"

	"github.com/jxskiss/base62"
)

const (
	totalLetters = 6
	encodedChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
)

type empty struct{}

var validCharSet map[rune]empty

var (
	encoder           = base62.NewEncoding(encodedChars)
	errInvalidLength  = errors.New("invalid length")
	errUnexpectedChar = errors.New("unexpected char")
)

// Generate returns a 6-letters id by given URL.
//
// TODO: extract this functionality to another stand-alone service
func Generate(url string) string {
	// padding with time.Now().UnixNano() to reduce collision probability if give same URL
	bytes := md5.Sum([]byte(fmt.Sprintf("%s%d", url, time.Now().UnixNano())))
	encoded := encoder.EncodeToString(bytes[:])
	return encoded[:totalLetters]
}

func getValidCharSet() map[rune]empty {
	if validCharSet != nil {
		return validCharSet
	}
	// lazy initialize encodedCharSet
	validCharSet := make(map[rune]empty, len(encodedChars))
	for _, c := range encodedChars {
		validCharSet[c] = empty{}
	}
	return validCharSet
}

func Validate(id string) error {
	if len(id) != totalLetters {
		return errInvalidLength
	}
	validChars := getValidCharSet()
	for _, r := range id {
		if _, ok := validChars[r]; !ok {
			return errUnexpectedChar
		}
	}
	return nil
}
