package admin_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/obrenoalvim/back-template-go/internal/admin"
	"github.com/obrenoalvim/back-template-go/internal/config"
	"github.com/obrenoalvim/back-template-go/internal/db"
	"github.com/obrenoalvim/back-template-go/internal/diagnostics"
	"github.com/obrenoalvim/back-template-go/internal/models"
)

// Guards against N+1: NotesWithOwners must keep running exactly one query
// regardless of how many notes exist. If someone swaps the Joins("Owner")
// eager load for a per-note lookup, this test goes red before it reaches
// production.
func TestNotesWithOwners_runs_exactly_one_query_regardless_of_row_count(t *testing.T) {
	cfg := config.Load()
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		cfg.DatabaseURL = url
	}
	require.NoError(t, db.Migrate(cfg.DatabaseURL))

	gormDB, err := db.Connect(cfg.DatabaseURL)
	require.NoError(t, err)

	counter := &diagnostics.QueryCounter{}
	require.NoError(t, gormDB.Use(counter))

	owner := models.User{Email: "owner-query-count@example.com", PasswordHash: "x"}
	require.NoError(t, gormDB.Create(&owner).Error)
	t.Cleanup(func() {
		gormDB.Unscoped().Where("owner_id = ?", owner.ID).Delete(&models.Note{})
		gormDB.Unscoped().Delete(&owner)
	})

	seedNotes := func(n int) {
		for i := 0; i < n; i++ {
			require.NoError(t, gormDB.Create(&models.Note{OwnerID: owner.ID, Title: "note", Content: "content"}).Error)
		}
	}

	seedNotes(4)
	counter.Reset()
	notes, err := admin.NotesWithOwners(gormDB)
	require.NoError(t, err)
	require.Len(t, notes, 4)
	for _, n := range notes {
		require.Equal(t, owner.Email, n.Owner.Email)
	}
	require.EqualValues(t, 1, counter.Count())

	seedNotes(4) // now 8 total
	counter.Reset()
	notes, err = admin.NotesWithOwners(gormDB)
	require.NoError(t, err)
	require.Len(t, notes, 8)
	require.EqualValues(t, 1, counter.Count(), "query count must stay constant as row count grows")
}
