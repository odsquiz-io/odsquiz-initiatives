package repositories

import (
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
)

const repositoryTestID = "a0000000-0000-4000-8000-000000000001"

type capturedStatement struct {
	sql  string
	vars []any
}

func dryRunRepository(t *testing.T) (*postsRepository, *[]capturedStatement) {
	t.Helper()

	db, err := gorm.Open(postgres.Open("host=localhost user=test dbname=test"), &gorm.Config{
		DryRun:               true,
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}

	statements := []capturedStatement{}
	err = db.Callback().Query().After("gorm:query").Register("posts:capture", func(tx *gorm.DB) {
		statements = append(statements, capturedStatement{sql: tx.Statement.SQL.String(), vars: tx.Statement.Vars})
	})
	if err != nil {
		t.Fatal(err)
	}

	return &postsRepository{DB: db}, &statements
}

func TestListPostsFiltersVisibleContentAndPagesByKeyset(t *testing.T) {
	repo, statements := dryRunRepository(t)
	cursor := &models.FeedCursor{CreatedAt: time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC), ID: repositoryTestID}
	initiativeID := "11111111-1111-4111-8111-111111111111"

	if _, err := repo.ListPosts(models.PostFilter{Limit: 21}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ListPosts(models.PostFilter{InitiativeID: initiativeID, Cursor: cursor, Limit: 21}); err != nil {
		t.Fatal(err)
	}

	if len(*statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(*statements))
	}
	feed, initiative := (*statements)[0], (*statements)[1]

	for _, statement := range []capturedStatement{feed, initiative} {
		for _, fragment := range []string{
			"JOIN initiatives AS i ON i.id::text = p.initiative_id::text",
			"JOIN users AS u ON u.id::text = p.author_id::text",
			visiblePost,
			"c.suspended_at IS NULL",
			"cu.suspended_at IS NULL",
			"ORDER BY p.created_at DESC, p.id DESC",
		} {
			if !strings.Contains(statement.sql, fragment) {
				t.Errorf("missing %q in %s", fragment, statement.sql)
			}
		}
		if !slices.Contains(statement.vars, any(21)) {
			t.Errorf("limit not bound: %v", statement.vars)
		}
	}

	if strings.Contains(feed.sql, "p.created_at, p.id) <") || strings.Contains(feed.sql, "p.initiative_id =") {
		t.Errorf("first feed page must not filter by cursor or initiative: %s", feed.sql)
	}
	if !strings.Contains(initiative.sql, "(p.created_at, p.id) < (") || !strings.Contains(initiative.sql, "p.initiative_id =") {
		t.Errorf("cursor or initiative filter missing: %s", initiative.sql)
	}
	if !slices.Contains(initiative.vars, any(initiativeID)) || !slices.Contains(initiative.vars, any(cursor.ID)) ||
		!slices.Contains(initiative.vars, any(cursor.CreatedAt)) {
		t.Errorf("filters not bound as parameters: %v", initiative.vars)
	}
}

func TestListCommentsFiltersVisibleContentOldestFirst(t *testing.T) {
	repo, statements := dryRunRepository(t)
	cursor := &models.FeedCursor{CreatedAt: time.Date(2026, 10, 22, 9, 0, 0, 0, time.UTC), ID: repositoryTestID}

	if _, err := repo.ListComments(models.CommentFilter{PostID: repositoryTestID, Cursor: cursor, Limit: 51}); err != nil {
		t.Fatal(err)
	}

	statement := (*statements)[0]
	for _, fragment := range []string{
		"JOIN users AS u ON u.id::text = c.author_id::text",
		visibleComment,
		"c.post_id =",
		"(c.created_at, c.id) > (",
		"ORDER BY c.created_at, c.id",
	} {
		if !strings.Contains(statement.sql, fragment) {
			t.Errorf("missing %q in %s", fragment, statement.sql)
		}
	}
}

func TestPostsSelectsMatchScanTargets(t *testing.T) {
	cases := map[string]struct {
		columns string
		target  any
	}{
		"posts":    {postColumns, &models.PostRecord{}},
		"comments": {commentColumns, &models.CommentRecord{}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			parsed, err := schema.Parse(tc.target, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatal(err)
			}

			selected := selectedNames(tc.columns)
			for _, column := range selected {
				if parsed.LookUpField(column) == nil {
					t.Errorf("selected column %q has no field", column)
				}
			}
			for _, field := range parsed.DBNames {
				if !slices.Contains(selected, field) {
					t.Errorf("field %q is never selected", field)
				}
			}
		})
	}
}

// selectedNames returns the result column names of a select list.
func selectedNames(columns string) []string {
	var names []string
	depth, start := 0, 0
	for index, char := range columns + "," {
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth > 0 {
				continue
			}
			column := strings.TrimSpace(columns[start:min(index, len(columns))])
			if position := strings.LastIndex(column, " AS "); position >= 0 {
				column = column[position+len(" AS "):]
			} else {
				column = column[strings.Index(column, ".")+1:]
			}
			names = append(names, column)
			start = index + 1
		}
	}
	return names
}
