package custom

import (
	"bufio"
	"strings"
)

func CustomSplit() func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	substring := "}"
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if i := strings.Index(string(data), substring); i >= 0 {
			return i + 2, data[0 : i+1], nil
		}

		if !atEOF {
			return 0, nil, nil
		}
		return len(data), data, bufio.ErrFinalToken
	}
}
