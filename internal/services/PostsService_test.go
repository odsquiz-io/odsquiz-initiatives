package services

import (
	"encoding/base64"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kauanpecanha/odsquiz-initiatives/internal/apperrors"
	"github.com/kauanpecanha/odsquiz-initiatives/internal/models"
	"gorm.io/gorm"
)

const (
	memberID  = "member"
	pendingID = "pending"

	activeInitiativeID = "11111111-1111-4111-8111-111111111111"
	hiddenInitiativeID = "22222222-2222-4222-8222-222222222222"
	missingID          = "99999999-9999-4999-8999-999999999999"

	ownerPostID       = "a0000000-0000-4000-8000-000000000001"
	memberPostID      = "a0000000-0000-4000-8000-000000000002"
	suspendedPostID   = "a0000000-0000-4000-8000-000000000003"
	hiddenPostID      = "a0000000-0000-4000-8000-000000000004"
	blockedAuthorPost = "a0000000-0000-4000-8000-000000000005"

	otherCommentID     = "c0000000-0000-4000-8000-000000000001"
	suspendedCommentID = "c0000000-0000-4000-8000-000000000002"
	blockedCommentID   = "c0000000-0000-4000-8000-000000000003"
	hiddenPostComment  = "c0000000-0000-4000-8000-000000000004"
)

var feedStart = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

type fakeUser struct {
	name        string
	suspendedAt *time.Time
}

type fakePostsRepository struct {
	initiatives map[string]models.Initiative
	members     map[[2]string]string
	users       map[string]fakeUser
	posts       map[string]models.InitiativePost
	comments    map[string]models.InitiativePostComment
	lastLimit   int
}

func (r *fakePostsRepository) ReadInitiative(id string) (*models.Initiative, error) {
	one, ok := r.initiatives[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return &one, nil
}

func (r *fakePostsRepository) IsActiveMember(initiativeID, userID string) (bool, error) {
	return r.members[[2]string{initiativeID, userID}] == "active", nil
}

func (r *fakePostsRepository) ListPosts(filter models.PostFilter) ([]models.PostRecord, error) {
	r.lastLimit = filter.Limit

	records := []models.PostRecord{}
	for _, post := range r.posts {
		record, ok := r.postRecord(post)
		if !ok || !postVisible(&record) || (filter.InitiativeID != "" && record.InitiativeID != filter.InitiativeID) {
			continue
		}
		if filter.Cursor != nil && !feedBefore(record.CreatedAt, record.ID, filter.Cursor.CreatedAt, filter.Cursor.ID) {
			continue
		}
		records = append(records, record)
	}

	slices.SortFunc(records, func(a, b models.PostRecord) int {
		return feedCompare(b.CreatedAt, b.ID, a.CreatedAt, a.ID)
	})

	return records[:min(len(records), filter.Limit)], nil
}

func (r *fakePostsRepository) ReadPost(id string) (*models.PostRecord, error) {
	record, ok := r.postRecord(r.posts[id])
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return &record, nil
}

func (r *fakePostsRepository) CreatePost(post *models.InitiativePost) error {
	r.posts[post.ID] = *post
	return nil
}

func (r *fakePostsRepository) UpdatePostBody(id, body string, editedAt time.Time) error {
	post, ok := r.posts[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}

	post.Body, post.EditedAt = body, &editedAt
	r.posts[id] = post
	return nil
}

func (r *fakePostsRepository) DeletePost(id string) error {
	if _, ok := r.posts[id]; !ok {
		return gorm.ErrRecordNotFound
	}

	delete(r.posts, id)
	for commentID, comment := range r.comments {
		if comment.PostID == id {
			delete(r.comments, commentID)
		}
	}
	return nil
}

func (r *fakePostsRepository) ListComments(filter models.CommentFilter) ([]models.CommentRecord, error) {
	r.lastLimit = filter.Limit

	records := []models.CommentRecord{}
	for _, comment := range r.comments {
		record, ok := r.commentRecord(comment)
		if !ok || !commentVisible(&record) || record.PostID != filter.PostID {
			continue
		}
		if filter.Cursor != nil && !feedBefore(filter.Cursor.CreatedAt, filter.Cursor.ID, record.CreatedAt, record.ID) {
			continue
		}
		records = append(records, record)
	}

	slices.SortFunc(records, func(a, b models.CommentRecord) int {
		return feedCompare(a.CreatedAt, a.ID, b.CreatedAt, b.ID)
	})

	return records[:min(len(records), filter.Limit)], nil
}

func (r *fakePostsRepository) ReadComment(id string) (*models.CommentRecord, error) {
	record, ok := r.commentRecord(r.comments[id])
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}

	return &record, nil
}

func (r *fakePostsRepository) CreateComment(comment *models.InitiativePostComment) error {
	r.comments[comment.ID] = *comment
	return nil
}

func (r *fakePostsRepository) UpdateCommentBody(id, body string, editedAt time.Time) error {
	comment, ok := r.comments[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}

	comment.Body, comment.EditedAt = body, &editedAt
	r.comments[id] = comment
	return nil
}

func (r *fakePostsRepository) DeleteComment(id string) error {
	if _, ok := r.comments[id]; !ok {
		return gorm.ErrRecordNotFound
	}

	delete(r.comments, id)
	return nil
}

func (r *fakePostsRepository) postRecord(post models.InitiativePost) (models.PostRecord, bool) {
	one, found := r.initiatives[post.InitiativeID]
	author, known := r.users[post.AuthorID]
	if post.ID == "" || !found || !known {
		return models.PostRecord{}, false
	}

	record := models.PostRecord{
		ID: post.ID, InitiativeID: post.InitiativeID, AuthorID: post.AuthorID, Body: post.Body,
		CreatedAt: post.CreatedAt, EditedAt: post.EditedAt, SuspendedAt: post.SuspendedAt,
		InitiativeName: one.Name, InitiativeOwner: one.Owner, InitiativeMainODS: one.MainODS,
		InitiativeCity: one.City, InitiativeState: one.State, InitiativeLogoObject: one.LogoObject,
		InitiativeSuspendedAt: one.SuspendedAt, AuthorName: author.name, AuthorSuspendedAt: author.suspendedAt,
	}
	for _, comment := range r.comments {
		if commentRecord, ok := r.commentRecord(comment); ok && comment.PostID == post.ID && commentVisible(&commentRecord) {
			record.CommentsCount++
		}
	}

	return record, true
}

func (r *fakePostsRepository) commentRecord(comment models.InitiativePostComment) (models.CommentRecord, bool) {
	author, known := r.users[comment.AuthorID]
	if comment.ID == "" || !known {
		return models.CommentRecord{}, false
	}

	return models.CommentRecord{
		ID: comment.ID, PostID: comment.PostID, AuthorID: comment.AuthorID, Body: comment.Body,
		CreatedAt: comment.CreatedAt, EditedAt: comment.EditedAt, SuspendedAt: comment.SuspendedAt,
		AuthorName: author.name, AuthorSuspendedAt: author.suspendedAt,
	}, true
}

func feedCompare(aTime time.Time, aID string, bTime time.Time, bID string) int {
	if compared := aTime.Compare(bTime); compared != 0 {
		return compared
	}

	return strings.Compare(aID, bID)
}

func feedBefore(aTime time.Time, aID string, bTime time.Time, bID string) bool {
	return feedCompare(aTime, aID, bTime, bID) < 0
}

func newPostsService() (*PostsService, *fakePostsRepository) {
	suspendedAt := feedStart.Add(-time.Hour)
	logo := "initiatives/" + activeInitiativeID + "/logo-0123456789abcdef.png"

	repo := &fakePostsRepository{
		initiatives: map[string]models.Initiative{
			activeInitiativeID: {ID: activeInitiativeID, Name: "Horta Viva", Owner: ownerID, MainODS: 2, City: "Niterói", State: "RJ", LogoObject: &logo},
			hiddenInitiativeID: {ID: hiddenInitiativeID, Name: "Suspensa", Owner: ownerID, MainODS: 13, SuspendedAt: &suspendedAt},
		},
		members: map[[2]string]string{
			{activeInitiativeID, memberID}:    "active",
			{activeInitiativeID, pendingID}:   "pending",
			{activeInitiativeID, suspendedID}: "active",
			{hiddenInitiativeID, memberID}:    "active",
		},
		users: map[string]fakeUser{
			ownerID:     {name: "Ana"},
			memberID:    {name: "Bruno"},
			pendingID:   {name: "Clara"},
			otherUserID: {name: "Davi"},
			suspendedID: {name: "Eva", suspendedAt: &suspendedAt},
		},
		posts:    map[string]models.InitiativePost{},
		comments: map[string]models.InitiativePostComment{},
	}

	seedPost(repo, ownerPostID, activeInitiativeID, ownerID, 1, nil)
	seedPost(repo, memberPostID, activeInitiativeID, memberID, 2, nil)
	seedPost(repo, suspendedPostID, activeInitiativeID, memberID, 3, &suspendedAt)
	seedPost(repo, hiddenPostID, hiddenInitiativeID, ownerID, 4, nil)
	seedPost(repo, blockedAuthorPost, activeInitiativeID, suspendedID, 5, nil)

	seedComment(repo, otherCommentID, memberPostID, otherUserID, 1, nil)
	seedComment(repo, suspendedCommentID, memberPostID, otherUserID, 2, &suspendedAt)
	seedComment(repo, blockedCommentID, memberPostID, suspendedID, 3, nil)
	seedComment(repo, hiddenPostComment, hiddenPostID, otherUserID, 4, nil)

	return &PostsService{Repo: repo, Initiatives: &Service{Repo: &fakeRepository{}}}, repo
}

func seedPost(repo *fakePostsRepository, id, initiativeID, authorID string, minute int, suspendedAt *time.Time) {
	createdAt := feedStart.Add(time.Duration(minute) * time.Minute)
	repo.posts[id] = models.InitiativePost{
		ID: id, InitiativeID: initiativeID, AuthorID: authorID, Body: "post " + id,
		CreatedAt: createdAt, UpdatedAt: createdAt, SuspendedAt: suspendedAt,
	}
}

func seedComment(repo *fakePostsRepository, id, postID, authorID string, minute int, suspendedAt *time.Time) {
	createdAt := feedStart.Add(time.Duration(minute) * time.Minute)
	repo.comments[id] = models.InitiativePostComment{
		ID: id, PostID: postID, AuthorID: authorID, Body: "comment " + id,
		CreatedAt: createdAt, UpdatedAt: createdAt, SuspendedAt: suspendedAt,
	}
}

func postIDs(page *models.PostPage) []string {
	ids := []string{}
	for _, item := range page.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func TestCreatePostNeedsOwnerOrActiveMember(t *testing.T) {
	tests := []struct {
		userID, initiativeID string
		want                 apperrors.Code
	}{
		{ownerID, activeInitiativeID, ""},
		{memberID, activeInitiativeID, ""},
		{pendingID, activeInitiativeID, apperrors.CodeForbidden},
		{otherUserID, activeInitiativeID, apperrors.CodeForbidden},
		{suspendedID, activeInitiativeID, apperrors.CodeAccountSuspended},
		{ownerID, hiddenInitiativeID, apperrors.CodeForbidden},
		{memberID, hiddenInitiativeID, apperrors.CodeInitiativeNotFound},
		{otherUserID, hiddenInitiativeID, apperrors.CodeInitiativeNotFound},
		{ownerID, missingID, apperrors.CodeInitiativeNotFound},
	}

	for _, test := range tests {
		service, repo := newPostsService()
		before := len(repo.posts)

		post, err := service.CreatePost(test.userID, test.initiativeID, "  Mutirão no sábado!\nTragam luvas.  ")
		assertCode(t, err, test.want)

		if created := len(repo.posts) > before; created != (test.want == "") {
			t.Fatalf("%s posting on %s: created = %v", test.userID, test.initiativeID, created)
		}
		if test.want != "" {
			continue
		}

		if post.Body != "Mutirão no sábado!\nTragam luvas." || post.Author.ID != test.userID || post.EditedAt != nil || post.CommentsCount != 0 {
			t.Fatalf("unexpected post %+v", post)
		}
		if !post.ViewerCanEdit || !post.ViewerCanDelete {
			t.Fatalf("author should edit and delete: %+v", post)
		}
		wantLogo := "/api/initiatives/" + activeInitiativeID + "/logo?v=0123456789abcdef"
		if post.Initiative.ID != activeInitiativeID || post.Initiative.Name != "Horta Viva" || post.Initiative.MainODS != 2 ||
			post.Initiative.LogoURL == nil || *post.Initiative.LogoURL != wantLogo {
			t.Fatalf("unexpected initiative summary %+v", post.Initiative)
		}
	}
}

func TestBodiesAreTrimmedAndBounded(t *testing.T) {
	tests := []struct {
		body       string
		maxLength  int
		want       string
		wantErrors bool
	}{
		{"  olá  ", maxPostBodyLength, "olá", false},
		{strings.Repeat("é", maxPostBodyLength), maxPostBodyLength, strings.Repeat("é", maxPostBodyLength), false},
		{" " + strings.Repeat("a", maxPostBodyLength) + "\n", maxPostBodyLength, strings.Repeat("a", maxPostBodyLength), false},
		{strings.Repeat("é", maxPostBodyLength+1), maxPostBodyLength, "", true},
		{strings.Repeat("a", maxCommentBodyLength+1), maxCommentBodyLength, "", true},
		{"", maxPostBodyLength, "", true},
		{" \n\t ", maxCommentBodyLength, "", true},
		{"a\x00b", maxPostBodyLength, "", true},
		{"a\xffb", maxPostBodyLength, "", true},
	}

	for _, test := range tests {
		body, err := validBody(test.body, test.maxLength)
		if test.wantErrors {
			assertCode(t, err, apperrors.CodeInvalidRequest)
			continue
		}

		assertCode(t, err, "")
		if body != test.want {
			t.Fatalf("validBody(%q) = %q, want %q", test.body, body, test.want)
		}
	}

	service, repo := newPostsService()
	before := len(repo.posts)

	_, err := service.CreatePost(ownerID, activeInitiativeID, strings.Repeat("a", maxPostBodyLength+1))
	assertCode(t, err, apperrors.CodeInvalidRequest)
	_, err = service.UpdatePost(memberID, memberPostID, "   ")
	assertCode(t, err, apperrors.CodeInvalidRequest)
	_, err = service.CreateComment(otherUserID, memberPostID, strings.Repeat("a", maxCommentBodyLength+1))
	assertCode(t, err, apperrors.CodeInvalidRequest)
	_, err = service.UpdateComment(otherUserID, otherCommentID, "")
	assertCode(t, err, apperrors.CodeInvalidRequest)

	if len(repo.posts) != before || repo.posts[memberPostID].Body != "post "+memberPostID {
		t.Fatal("invalid bodies must not be stored")
	}
}

func TestOnlyTheAuthorEditsAPost(t *testing.T) {
	tests := []struct {
		userID, postID string
		want           apperrors.Code
	}{
		{memberID, memberPostID, ""},
		{ownerID, ownerPostID, ""},
		{ownerID, memberPostID, apperrors.CodeForbidden},
		{otherUserID, memberPostID, apperrors.CodeForbidden},
		{memberID, suspendedPostID, apperrors.CodePostNotFound},
		{ownerID, hiddenPostID, apperrors.CodePostNotFound},
		{suspendedID, blockedAuthorPost, apperrors.CodeAccountSuspended},
		{memberID, missingID, apperrors.CodePostNotFound},
	}

	for _, test := range tests {
		service, repo := newPostsService()

		post, err := service.UpdatePost(test.userID, test.postID, " novo texto ")
		assertCode(t, err, test.want)

		stored := repo.posts[test.postID]
		if edited := stored.Body == "novo texto"; edited != (test.want == "") {
			t.Fatalf("%s editing %s: edited = %v", test.userID, test.postID, edited)
		}
		if test.want == "" && (post.Body != "novo texto" || post.EditedAt == nil || stored.EditedAt == nil) {
			t.Fatalf("edit must set the body and edited_at: %+v", post)
		}
	}
}

func TestEditingWithTheSameBodyKeepsEditedAtEmpty(t *testing.T) {
	service, repo := newPostsService()

	post, err := service.UpdatePost(memberID, memberPostID, "  post "+memberPostID+" ")
	assertCode(t, err, "")
	if post.EditedAt != nil || repo.posts[memberPostID].EditedAt != nil {
		t.Fatal("unchanged body must not mark the post as edited")
	}

	comment, err := service.UpdateComment(otherUserID, otherCommentID, "comment "+otherCommentID)
	assertCode(t, err, "")
	if comment.EditedAt != nil || repo.comments[otherCommentID].EditedAt != nil {
		t.Fatal("unchanged body must not mark the comment as edited")
	}
}

func TestPostAuthorOrInitiativeOwnerDeletesAPost(t *testing.T) {
	tests := []struct {
		userID, postID string
		want           apperrors.Code
	}{
		{memberID, memberPostID, ""},
		{ownerID, memberPostID, ""},
		{ownerID, ownerPostID, ""},
		{memberID, ownerPostID, apperrors.CodeForbidden},
		{otherUserID, memberPostID, apperrors.CodeForbidden},
		{ownerID, suspendedPostID, apperrors.CodePostNotFound},
		{ownerID, hiddenPostID, apperrors.CodePostNotFound},
		{suspendedID, blockedAuthorPost, apperrors.CodeAccountSuspended},
		{ownerID, missingID, apperrors.CodePostNotFound},
	}

	for _, test := range tests {
		service, repo := newPostsService()

		err := service.DeletePost(test.userID, test.postID)
		assertCode(t, err, test.want)

		if _, kept := repo.posts[test.postID]; kept == (test.want == "") && test.postID != missingID {
			t.Fatalf("%s deleting %s: kept = %v", test.userID, test.postID, kept)
		}
	}
}

func TestCommentRules(t *testing.T) {
	service, repo := newPostsService()

	comment, err := service.CreateComment(otherUserID, memberPostID, " Vou levar mudas. ")
	assertCode(t, err, "")
	if comment.Body != "Vou levar mudas." || comment.PostID != memberPostID || comment.Author.Name != "Davi" ||
		!comment.ViewerCanEdit || !comment.ViewerCanDelete {
		t.Fatalf("unexpected comment %+v", comment)
	}
	if _, stored := repo.comments[comment.ID]; !stored {
		t.Fatal("comment was not stored")
	}

	tests := []struct {
		userID, postID string
		want           apperrors.Code
	}{
		{pendingID, ownerPostID, ""},
		{suspendedID, memberPostID, apperrors.CodeAccountSuspended},
		{otherUserID, suspendedPostID, apperrors.CodePostNotFound},
		{otherUserID, hiddenPostID, apperrors.CodePostNotFound},
		{otherUserID, blockedAuthorPost, apperrors.CodePostNotFound},
		{otherUserID, missingID, apperrors.CodePostNotFound},
	}

	for _, test := range tests {
		_, err := service.CreateComment(test.userID, test.postID, "comentário")
		assertCode(t, err, test.want)
	}
}

func TestOnlyTheAuthorEditsAComment(t *testing.T) {
	tests := []struct {
		userID, commentID string
		want              apperrors.Code
	}{
		{otherUserID, otherCommentID, ""},
		{ownerID, otherCommentID, apperrors.CodeForbidden},
		{memberID, otherCommentID, apperrors.CodeForbidden},
		{otherUserID, suspendedCommentID, apperrors.CodeCommentNotFound},
		{otherUserID, hiddenPostComment, apperrors.CodeCommentNotFound},
		{suspendedID, blockedCommentID, apperrors.CodeAccountSuspended},
		{otherUserID, missingID, apperrors.CodeCommentNotFound},
	}

	for _, test := range tests {
		service, repo := newPostsService()

		comment, err := service.UpdateComment(test.userID, test.commentID, "corrigido")
		assertCode(t, err, test.want)

		if edited := repo.comments[test.commentID].Body == "corrigido"; edited != (test.want == "") {
			t.Fatalf("%s editing %s: edited = %v", test.userID, test.commentID, edited)
		}
		if test.want == "" && comment.EditedAt == nil {
			t.Fatal("edit must set edited_at")
		}
	}
}

func TestCommentAuthorOrInitiativeOwnerDeletesAComment(t *testing.T) {
	tests := []struct {
		userID, commentID string
		want              apperrors.Code
	}{
		{otherUserID, otherCommentID, ""},
		{ownerID, otherCommentID, ""},
		{memberID, otherCommentID, apperrors.CodeForbidden},
		{pendingID, otherCommentID, apperrors.CodeForbidden},
		{ownerID, suspendedCommentID, apperrors.CodeCommentNotFound},
		{ownerID, blockedCommentID, apperrors.CodeCommentNotFound},
		{ownerID, hiddenPostComment, apperrors.CodeCommentNotFound},
		{suspendedID, blockedCommentID, apperrors.CodeAccountSuspended},
		{ownerID, missingID, apperrors.CodeCommentNotFound},
	}

	for _, test := range tests {
		service, repo := newPostsService()

		err := service.DeleteComment(test.userID, test.commentID)
		assertCode(t, err, test.want)

		if _, kept := repo.comments[test.commentID]; kept == (test.want == "") && test.commentID != missingID {
			t.Fatalf("%s deleting %s: kept = %v", test.userID, test.commentID, kept)
		}
	}
}

func TestFeedShowsOnlyVisiblePostsWithViewerPermissions(t *testing.T) {
	service, _ := newPostsService()

	page, err := service.ListFeed(ownerID, "", 0)
	assertCode(t, err, "")

	if ids := postIDs(page); !slices.Equal(ids, []string{memberPostID, ownerPostID}) || page.NextCursor != nil {
		t.Fatalf("feed = %v, next = %v", ids, page.NextCursor)
	}

	memberPost := page.Items[0]
	if memberPost.ViewerCanEdit || !memberPost.ViewerCanDelete || memberPost.CommentsCount != 1 {
		t.Fatalf("owner viewing a member post: %+v", memberPost)
	}

	page, err = service.ListFeed(memberID, "", 0)
	assertCode(t, err, "")
	if !page.Items[0].ViewerCanEdit || !page.Items[0].ViewerCanDelete || page.Items[1].ViewerCanEdit || page.Items[1].ViewerCanDelete {
		t.Fatalf("member permissions: %+v", page.Items)
	}

	page, err = service.ListFeed(otherUserID, "", 0)
	assertCode(t, err, "")
	for _, item := range page.Items {
		if item.ViewerCanEdit || item.ViewerCanDelete {
			t.Fatalf("stranger permissions: %+v", item)
		}
	}
}

func TestInitiativePostsHideSuspendedInitiatives(t *testing.T) {
	service, _ := newPostsService()

	page, err := service.ListInitiativePosts(otherUserID, activeInitiativeID, "", 0)
	assertCode(t, err, "")
	if ids := postIDs(page); !slices.Equal(ids, []string{memberPostID, ownerPostID}) {
		t.Fatalf("initiative posts = %v", ids)
	}

	page, err = service.ListInitiativePosts(ownerID, hiddenInitiativeID, "", 0)
	assertCode(t, err, "")
	if page.Items == nil || len(page.Items) != 0 || page.NextCursor != nil {
		t.Fatalf("owner of a suspended initiative must get an empty list, got %+v", page)
	}

	_, err = service.ListInitiativePosts(memberID, hiddenInitiativeID, "", 0)
	assertCode(t, err, apperrors.CodeInitiativeNotFound)
	_, err = service.ListInitiativePosts(otherUserID, missingID, "", 0)
	assertCode(t, err, apperrors.CodeInitiativeNotFound)
	_, err = service.ListInitiativePosts(otherUserID, activeInitiativeID, "not-a-cursor", 0)
	assertCode(t, err, apperrors.CodeInvalidRequest)
}

func TestCommentListHidesSuspendedContent(t *testing.T) {
	service, _ := newPostsService()

	page, err := service.ListComments(ownerID, memberPostID, "", 0)
	assertCode(t, err, "")
	if len(page.Items) != 1 || page.Items[0].ID != otherCommentID || page.NextCursor != nil {
		t.Fatalf("comments = %+v", page)
	}
	if comment := page.Items[0]; comment.ViewerCanEdit || !comment.ViewerCanDelete {
		t.Fatalf("initiative owner permissions on a comment: %+v", comment)
	}

	for _, postID := range []string{suspendedPostID, hiddenPostID, blockedAuthorPost, missingID} {
		_, err := service.ListComments(ownerID, postID, "", 0)
		assertCode(t, err, apperrors.CodePostNotFound)
	}
}

func TestFeedCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 10, 22, 9, 30, 15, 123456000, time.FixedZone("BRT", -3*60*60))
	id := "A0000000-0000-4000-8000-0000000000FF"

	cursor := encodeFeedCursor(createdAt, strings.ToLower(id))
	if strings.ContainsAny(cursor, "+/=") {
		t.Fatalf("cursor %q is not unpadded base64url", cursor)
	}

	decoded, err := decodeFeedCursor(cursor)
	assertCode(t, err, "")
	if !decoded.CreatedAt.Equal(createdAt) || decoded.ID != strings.ToLower(id) {
		t.Fatalf("decoded = %+v", decoded)
	}

	padded := base64.URLEncoding.EncodeToString([]byte(createdAt.Format(time.RFC3339Nano) + "|" + id))
	decoded, err = decodeFeedCursor(padded)
	assertCode(t, err, "")
	if decoded.ID != strings.ToLower(id) {
		t.Fatalf("decoded id = %q, want canonical lowercase", decoded.ID)
	}

	if decoded, err := decodeFeedCursor(""); decoded != nil || err != nil {
		t.Fatalf("empty cursor = %v, %v", decoded, err)
	}

	for _, invalid := range []string{
		"%%%",
		base64.RawURLEncoding.EncodeToString([]byte("no separator")),
		base64.RawURLEncoding.EncodeToString([]byte("yesterday|" + ownerPostID)),
		base64.RawURLEncoding.EncodeToString([]byte(createdAt.Format(time.RFC3339Nano) + "|not-a-uuid")),
		base64.RawURLEncoding.EncodeToString([]byte(createdAt.Format(time.RFC3339Nano) + "|")),
	} {
		_, err := decodeFeedCursor(invalid)
		assertCode(t, err, apperrors.CodeInvalidRequest)
	}
}

func TestFeedPagesFollowTheCursor(t *testing.T) {
	service, repo := newPostsService()
	tied := feedStart.Add(10 * time.Minute)
	for _, id := range []string{"b0000000-0000-4000-8000-000000000001", "b0000000-0000-4000-8000-000000000002"} {
		repo.posts[id] = models.InitiativePost{ID: id, InitiativeID: activeInitiativeID, AuthorID: ownerID, Body: "tied", CreatedAt: tied}
	}
	want := []string{"b0000000-0000-4000-8000-000000000002", "b0000000-0000-4000-8000-000000000001", memberPostID, ownerPostID}

	var got []string
	cursor := ""
	for pages := 1; ; pages++ {
		page, err := service.ListFeed(otherUserID, cursor, 3)
		assertCode(t, err, "")
		if repo.lastLimit != 4 {
			t.Fatalf("repository limit = %d, want limit + 1", repo.lastLimit)
		}

		got = append(got, postIDs(page)...)
		if page.NextCursor == nil {
			if pages != 2 {
				t.Fatalf("got %d pages, want 2", pages)
			}
			break
		}
		cursor = *page.NextCursor
	}
	if !slices.Equal(got, want) {
		t.Fatalf("paged feed = %v, want %v", got, want)
	}

	page, err := service.ListFeed(otherUserID, "", len(want))
	assertCode(t, err, "")
	if len(page.Items) != len(want) || page.NextCursor != nil {
		t.Fatalf("a page holding every post must not return a cursor: %+v", page.NextCursor)
	}

	_, err = service.ListFeed(otherUserID, "", 0)
	assertCode(t, err, "")
	if repo.lastLimit != defaultPostsLimit+1 {
		t.Fatalf("default limit = %d", repo.lastLimit-1)
	}

	_, err = service.ListFeed(otherUserID, "", 500)
	assertCode(t, err, "")
	if repo.lastLimit != maxPostsLimit+1 {
		t.Fatalf("capped limit = %d", repo.lastLimit-1)
	}
}

func TestCommentPagesAreOldestFirst(t *testing.T) {
	service, repo := newPostsService()
	for minute, id := range []string{"d0000000-0000-4000-8000-000000000001", "d0000000-0000-4000-8000-000000000002"} {
		seedComment(repo, id, memberPostID, memberID, 10+minute, nil)
	}
	want := []string{otherCommentID, "d0000000-0000-4000-8000-000000000001", "d0000000-0000-4000-8000-000000000002"}

	first, err := service.ListComments(otherUserID, memberPostID, "", 2)
	assertCode(t, err, "")
	if first.NextCursor == nil || len(first.Items) != 2 {
		t.Fatalf("first page = %+v", first)
	}

	second, err := service.ListComments(otherUserID, memberPostID, *first.NextCursor, 2)
	assertCode(t, err, "")
	if second.NextCursor != nil {
		t.Fatal("last page must not return a cursor")
	}

	var got []string
	for _, item := range append(first.Items, second.Items...) {
		got = append(got, item.ID)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("paged comments = %v, want %v", got, want)
	}

	_, err = service.ListComments(otherUserID, memberPostID, "", 0)
	assertCode(t, err, "")
	if repo.lastLimit != defaultCommentsLimit+1 {
		t.Fatalf("default limit = %d", repo.lastLimit-1)
	}

	_, err = service.ListComments(otherUserID, memberPostID, "", 1000)
	assertCode(t, err, "")
	if repo.lastLimit != maxCommentsLimit+1 {
		t.Fatalf("capped limit = %d", repo.lastLimit-1)
	}
}
