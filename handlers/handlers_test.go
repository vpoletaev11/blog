package handlers_test

import (
	"blog/handlers"
	"bytes"
	"database/sql/driver"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlxmock "github.com/zhashkevych/go-sqlxmock"
)

type anyTime struct{}

// ()Match() checks if input value is time
func (a anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	if !ok {
		return false
	}
	return ok
}

func TestAddPostSuccess(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO post").WithArgs("title", "body", anyTime{}).WillReturnRows(sqlxmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("INSERT INTO tag").WithArgs("tag1", 1, "tag2", 1).WillReturnResult(sqlxmock.NewResult(2, 2))
	mock.ExpectCommit()

	postJSON := `{
		"title": "title",
		"body": "body",
		"tags": ["tag1", "tag2"]
	}`
	r := httptest.NewRequest("POST", "/posts", strings.NewReader(postJSON))
	w := httptest.NewRecorder()

	sut := handlers.AddPostJSON(db)
	sut(w, r)

	assertBodyEqual(t, "", w.Body)
}

func TestAddPostInsertPostDBError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO post").WithArgs("title", "body", anyTime{}).WillReturnError(fmt.Errorf("test db error"))
	mock.ExpectCommit()

	postJSON := `{
		"title": "title",
		"body": "body",
		"tags": ["tag1", "tag2"]
	}`
	r := httptest.NewRequest("POST", "/posts", strings.NewReader(postJSON))
	w := httptest.NewRecorder()

	sut := handlers.AddPostJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertBodyEqual(t, `{"error":"error while insert post into db: test db error"}`, w.Body)
}

func TestAddPostInsertTagsDBError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO post").WithArgs("title", "body", anyTime{}).WillReturnRows(sqlxmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("INSERT INTO tag").WithArgs("tag1", 1, "tag2", 1).WillReturnError(fmt.Errorf("test db error"))
	mock.ExpectRollback()

	postJSON := `{
		"title": "title",
		"body": "body",
		"tags": ["tag1", "tag2"]
	}`
	r := httptest.NewRequest("POST", "/posts", strings.NewReader(postJSON))
	w := httptest.NewRecorder()

	sut := handlers.AddPostJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertBodyEqual(t, `{"error":"error while insert post tags into db: test db error"}`, w.Body)
}

func TestListPostsSuccess(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	firstPostTimestamp, err := time.Parse("2006-01-02T15:04:05-00:00", "2022-03-10T12:43:12-00:00")
	require.NoError(t, err)
	secondPostTimestamp, err := time.Parse("2006-01-02T15:04:05-00:00", "2022-01-02T15:04:05-00:00")
	require.NoError(t, err)

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, title, body, created_at FROM post ORDER BY created_at DESC")).WithArgs(0, handlers.MaxPostsLimit).WillReturnRows(
		sqlxmock.NewRows([]string{
			"id", "title", "body", "created_at",
		}).AddRow(
			2, "title", "body", firstPostTimestamp,
		).AddRow(
			1, "title", "body", secondPostTimestamp,
		),
	)
	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, name, post_id FROM tag WHERE post_id IN")).WithArgs(2, 1).WillReturnRows(
		sqlxmock.NewRows([]string{
			"id", "name", "post_id",
		}).AddRow(
			1, "tag1", 1,
		).AddRow(
			2, "tag2", 1,
		).AddRow(
			3, "tag3", 2,
		),
	)

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assertBodyEqual(t, `[{"id":2,"title":"title","body":"body","created_at":"2022-03-10T12:43:12Z","tags":["tag3"]},{"id":1,"title":"title","body":"body","created_at":"2022-01-02T15:04:05Z","tags":["tag1","tag2"]}]`, w.Body)
}

func TestListPostsSelectPostsDBError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, title, body, created_at FROM post ORDER BY created_at DESC")).WillReturnError(fmt.Errorf("test db error"))

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertBodyEqual(t, `{"error":"error while fetch list of posts from db: test db error"}`, w.Body)
}

func TestListPostsSelectTagsError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id, title, body, created_at FROM post ORDER BY created_at DESC")).WithArgs(0, handlers.MaxPostsLimit).WillReturnRows(
		sqlxmock.NewRows([]string{
			"id", "title", "body",
		}).AddRow(
			1, "title", "body",
		).AddRow(
			2, "title", "body",
		),
	)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, post_id FROM tag WHERE post_id IN")).WillReturnError(fmt.Errorf("test db error"))

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertBodyEqual(t, `{"error":"error while fetch list of posts tags from db: test db error"}`, w.Body)
}

func TestListPostsIncorrectLimitParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts?offset=0&limit=wrong_value", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(nil)
	sut(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertBodyEqual(t, `{"error":"incorrect limit URL parameter, actual: wrong_value, expected: positive integer"}`, w.Body)
}

func TestListPostsNegativeLimitParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts?offset=0&limit=-20", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(nil)
	sut(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertBodyEqual(t, `{"error":"limit cannot be less than 0"}`, w.Body)
}

func TestListPostsOutOfBoundsLimitParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts?offset=0&limit=2000", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(nil)
	sut(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertBodyEqual(t, `{"error":"limit cannot be greater than 1000"}`, w.Body)
}

func TestListPostsIncorrectOffsetParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts?offset=wrong_value&limit=10", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(nil)
	sut(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertBodyEqual(t, `{"error":"incorrect offset URL parameter, actual: wrong_value, expected: positive integer"}`, w.Body)
}

func TestListPostsNegativeOffsetParam(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts?offset=-1&limit=100", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(nil)
	sut(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertBodyEqual(t, `{"error":"offset cannot be less than 0"}`, w.Body)
}

func assertBodyEqual(t *testing.T, expected string, actual *bytes.Buffer) {
	bodyBytes, err := ioutil.ReadAll(actual)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	assert.Equal(t, expected, bodyString)
}
