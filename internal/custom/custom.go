package custom

import (
	"bufio"
	"strings"

	"github.com/dcaiman/YP_GO/internal/clog"
)

func CustomSplit() func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		begin := strings.Index(string(data), "{")
		end := strings.Index(string(data), "}")
		if begin >= 0 && end >= 0 {
			return end + 2, data[begin : end+1], nil
		}

		if !atEOF || string(data) == "" {
			return 0, nil, nil
		}
		return len(data), data, clog.ToLog(clog.FuncName(), bufio.ErrFinalToken)
	}
}
