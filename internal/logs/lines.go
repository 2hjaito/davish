package logs

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// MaxLine is the longest line kept; the rest of a longer one is dropped,
// so a binary file or a minified blob cannot blow up the webview.
const MaxLine = 16 << 10

// readLines calls fn with each line of r, without its line ending, until
// r ends.
func readLines(r io.Reader, fn func(line string)) error {
	br := bufio.NewReaderSize(r, 64<<10)
	var long []byte
	for {
		chunk, err := br.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			if len(long) < MaxLine {
				long = append(long, chunk[:min(len(chunk), MaxLine-len(long))]...)
			}
			continue
		}
		if len(chunk) > 0 || len(long) > 0 {
			if len(long) < MaxLine {
				long = append(long, chunk[:min(len(chunk), MaxLine-len(long))]...)
			}
			fn(strings.TrimRight(string(long), "\r\n"))
			long = long[:0]
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// needsRoot tells whether a stderr line is the log tool itself saying the
// user may not read kind's source, so following it would show nothing
// forever. Only the tool's own wording counts: a container's stderr is its
// output, and an app may well log "Permission denied".
func needsRoot(kind, line string) bool {
	switch kind {
	case KindFile:
		return strings.HasPrefix(line, "tail: ") && strings.Contains(line, "Permission denied")
	case KindDocker:
		// Older clients say "Docker daemon socket", newer "docker API".
		return strings.Contains(strings.ToLower(line), "permission denied while trying to connect to the docker")
	case KindJournal:
		return strings.Contains(line, "not seeing messages from other users") ||
			strings.Contains(line, "insufficient permissions")
	}
	return false
}
