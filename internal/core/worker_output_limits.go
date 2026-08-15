package core

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxWorkerStreamCaptureBytes    = 4 << 20
	maxPersistedWorkerReportBytes = 128 << 10
	maxWorkerControlTextBytes      = 8 << 10
)

// boundedWorkerCapture keeps the start of a process stream plus a rolling tail.
// Workbench control markers are intentionally final lines, so retaining only a
// prefix would be unsafe: a noisy worker could push ATTENTION_REQUIRED or
// WORKER_UNAVAILABLE beyond the cap. Write always reports the original byte
// count so the child process is never back-pressured by Workbench truncation.
type boundedWorkerCapture struct {
	limit       int
	prefixLimit int
	tailLimit   int
	prefix      []byte
	tail        []byte
	total       int64
}

func newBoundedWorkerCapture(limit int) *boundedWorkerCapture {
	if limit < 0 {
		limit = 0
	}
	prefixLimit := limit / 2
	return &boundedWorkerCapture{
		limit:       limit,
		prefixLimit: prefixLimit,
		tailLimit:   limit - prefixLimit,
	}
}

func (c *boundedWorkerCapture) Write(p []byte) (int, error) {
	original := len(p)
	c.total += int64(original)
	if c.limit == 0 || original == 0 {
		return original, nil
	}

	if len(c.prefix) < c.prefixLimit {
		n := c.prefixLimit - len(c.prefix)
		if n > len(p) {
			n = len(p)
		}
		c.prefix = append(c.prefix, p[:n]...)
		p = p[n:]
	}
	if len(p) > 0 && c.tailLimit > 0 {
		c.appendTail(p)
	}
	return original, nil
}

func (c *boundedWorkerCapture) appendTail(p []byte) {
	if len(p) >= c.tailLimit {
		c.tail = append(c.tail[:0], p[len(p)-c.tailLimit:]...)
		return
	}
	over := len(c.tail) + len(p) - c.tailLimit
	if over > 0 {
		copy(c.tail, c.tail[over:])
		c.tail = c.tail[:len(c.tail)-over]
	}
	c.tail = append(c.tail, p...)
}

func (c *boundedWorkerCapture) Truncated() bool {
	return c.total > int64(len(c.prefix)+len(c.tail))
}

func (c *boundedWorkerCapture) String() string {
	if c == nil {
		return ""
	}
	if !c.Truncated() {
		out := make([]byte, 0, len(c.prefix)+len(c.tail))
		out = append(out, c.prefix...)
		out = append(out, c.tail...)
		return string(out)
	}
	omitted := c.total - int64(len(c.prefix)+len(c.tail))
	marker := fmt.Sprintf("\n\n[Workbench truncated %d bytes of worker stream output]\n\n", omitted)
	out := make([]byte, 0, len(c.prefix)+len(marker)+len(c.tail))
	out = append(out, c.prefix...)
	out = append(out, marker...)
	out = append(out, c.tail...)
	return string(out)
}

// boundPersistedWorkerText is the final durable-state defence. It is separate
// from live capture because normalized provider output can still be verbose and
// because remote RunnerResponse values are persisted outside Engine task state.
func boundPersistedWorkerText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxPersistedWorkerReportBytes {
		return text
	}
	reserve := 256
	budget := maxPersistedWorkerReportBytes - reserve
	if budget < 2 {
		budget = maxPersistedWorkerReportBytes
	}
	prefixBudget := budget / 2
	tailBudget := budget - prefixBudget
	prefix := utf8Prefix(text, prefixBudget)
	tail := utf8Tail(text, tailBudget)
	omitted := len(text) - len(prefix) - len(tail)
	if omitted < 0 {
		omitted = 0
	}
	marker := fmt.Sprintf("\n\n[Workbench truncated %d bytes before persisting this worker report]\n\n", omitted)
	return strings.TrimSpace(prefix + marker + tail)
}

func boundWorkerControlText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxWorkerControlTextBytes {
		return text
	}
	prefix := utf8Prefix(text, maxWorkerControlTextBytes-64)
	return strings.TrimSpace(prefix) + " … [truncated by Workbench]"
}

func boundRunResultForPersistence(res RunResult) RunResult {
	res.Output = boundPersistedWorkerText(res.Output)
	res.Attention = boundWorkerControlText(res.Attention)
	res.WorkerUnavailable = boundWorkerControlText(res.WorkerUnavailable)
	return res
}

func utf8Prefix(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	if len(text) <= limit {
		return text
	}
	end := limit
	for end > 0 && !utf8.ValidString(text[:end]) {
		end--
	}
	return text[:end]
}

func utf8Tail(text string, limit int) string {
	if limit <= 0 || text == "" {
		return ""
	}
	if len(text) <= limit {
		return text
	}
	start := len(text) - limit
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	return text[start:]
}
