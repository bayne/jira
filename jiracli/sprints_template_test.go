package jiracli

import (
	"bytes"
	"testing"

	"github.com/go-jira/jira/jiradata"
)

func TestSprintsTemplateRenders(t *testing.T) {
	data := jiradata.SprintResults{
		Values: []jiradata.Sprint{
			{Id: 101, Name: "Sprint 41", State: "closed", StartDate: "2026-05-15T07:00:00.000Z", EndDate: "2026-05-28T07:00:00.000Z", Goal: "Ship the thing"},
			{Id: 102, Name: "Sprint 42", State: "active", StartDate: "2026-05-29T07:00:00.000Z", EndDate: "2026-06-11T07:00:00.000Z"},
			{Id: 103, Name: "Sprint 43", State: "future"},
		},
	}
	buf := bytes.Buffer{}
	if err := RunTemplate("sprints", data, &buf); err != nil {
		t.Fatal(err)
	}
	t.Logf("\n%s", buf.String())
}
