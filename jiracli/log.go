package jiracli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	logging "gopkg.in/op/go-logging.v1"
)

var (
	log = logging.MustGetLogger("jira")

	// RunID is a per-invocation correlation identifier. Including it in
	// every log line lets operators trace one CLI run end-to-end through
	// central log aggregation.
	RunID = newRunID()
)

func newRunID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "norunid"
	}
	return hex.EncodeToString(b)
}

func IncreaseLogLevel(verbosity int) {
	logging.SetLevel(logging.GetLevel("")+logging.Level(verbosity), "")
}

func InitLogging() {
	logBackend := logging.NewLogBackend(os.Stderr, "", 0)
	format := os.Getenv("JIRA_LOG_FORMAT")
	if format == "" {
		format = "%{color}%{level:-5s}%{color:reset} %{message}"
	}
	logging.SetBackend(
		logging.NewBackendFormatter(
			logBackend,
			logging.MustStringFormatter(format),
		),
	)
	if os.Getenv("JIRA_DEBUG") == "" {
		logging.SetLevel(logging.NOTICE, "")
	} else {
		logging.SetLevel(logging.DEBUG, "")
		if verbosity, err := strconv.Atoi(os.Getenv("JIRA_DEBUG")); err == nil {
			IncreaseLogLevel(verbosity)
		}
	}
}

// logContext returns the structured prefix used by retry/validate code
// paths. It is intentionally key=value style so log aggregators can index
// each field; values are quoted only when needed.
func logContext(opts *GlobalOptions) string {
	app := "go-jira"
	env := "nonprod"
	if opts != nil {
		if v := opts.AppName.Value; v != "" {
			app = v
		}
		if v := opts.Environment.Value; v != "" {
			env = v
		}
	}
	return fmt.Sprintf("integration=%s env=%s run_id=%s", app, env, RunID)
}
