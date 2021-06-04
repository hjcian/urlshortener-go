package urlshortener

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/jxskiss/base62"
)

const totalLetters = 6

// Get returns a 6-letters id by given URL.
//
// TODO: extract this functionality to another stand-alone service
func Get(url string) string {
	// padding with time.Now().UnixNano() to reduce collision probability if give same URL
	bytes := md5.Sum([]byte(fmt.Sprintf("%s%d", url, time.Now().UnixNano())))
	encoded := base62.EncodeToString(bytes[:])
	return encoded[:totalLetters]
}
