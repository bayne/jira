package jiracmd

import (
	"testing"
	"time"

	"github.com/go-jira/jira/jiradata"
)

func TestPreviousSprints(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	sprints := []jiradata.Sprint{
		{Id: 1, State: "closed", StartDate: "2026-05-01T00:00:00.000Z", EndDate: "2026-05-14T00:00:00.000Z"},
		{Id: 2, State: "closed", StartDate: "2026-05-15T00:00:00.000Z", EndDate: "2026-05-28T00:00:00.000Z"},
		// the active (current) sprint must be excluded so offset 0 is the
		// previous sprint, not the current one
		{Id: 3, State: "active", StartDate: "2026-05-29T00:00:00.000Z", EndDate: "2026-06-09T00:00:00.000Z"},
		{Id: 4, State: "closed", StartDate: "2026-06-01T00:00:00.000Z", EndDate: "2026-06-15T00:00:00.000Z"},
		// future sprints must be excluded
		{Id: 5, State: "future", StartDate: "2026-06-16T00:00:00.000Z", EndDate: "2026-06-30T00:00:00.000Z"},
		// missing start date must be skipped, not selected
		{Id: 6, State: "closed"},
	}

	previous := previousSprints(sprints, now)

	if len(previous) != 3 {
		t.Fatalf("expected 3 previous sprints, got %d", len(previous))
	}
	for i, expected := range []int{4, 2, 1} {
		if previous[i].sprint.Id != expected {
			t.Errorf("expected sprint %d at position %d, got %d", expected, i, previous[i].sprint.Id)
		}
	}
}
