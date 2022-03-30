package handlers_test

import (
	"blog/handlers"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sqlxmock "github.com/zhashkevych/go-sqlxmock"
)

func TestAddPostSuccess(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO post").WithArgs("title", "body").WillReturnResult(sqlxmock.NewResult(1, 1))
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
	mock.ExpectExec("INSERT INTO post").WithArgs("title", "body").WillReturnError(fmt.Errorf("test db error"))
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
	assertBodyEqual(t, `{"error":"test db error"}`, w.Body)
}

func TestAddPostInsertTagsDBError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO post").WithArgs("title", "body").WillReturnResult(sqlxmock.NewResult(1, 1))
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
	assertBodyEqual(t, `{"error":"test db error"}`, w.Body)
}

func TestListPostsSuccess(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT (id, title, body) FROM post")).WithArgs(0, handlers.MaxPostsLimit).WillReturnRows(
		sqlxmock.NewRows([]string{
			"id", "title", "body",
		}).AddRow(
			1, "title", "body",
		).AddRow(
			2, "title", "body",
		),
	)
	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT (name, post_id) FROM tag WHERE post_id IN")).WithArgs(1, 2).WillReturnRows(
		sqlxmock.NewRows([]string{
			"name", "post_id",
		}).AddRow(
			"tag1", 1,
		).AddRow(
			"tag2", 1,
		).AddRow(
			"tag3", 2,
		),
	)

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assertBodyEqual(t, `[{"ID":1,"title":"title","body":"body","tags":["tag1","tag2"]},{"ID":2,"title":"title","body":"body","tags":["tag3"]}]`, w.Body)
}

func TestListPostsSelectPostsDBError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT (id, title, body) FROM post")).WillReturnError(fmt.Errorf("test db error"))

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertBodyEqual(t, `{"error":"test db error"}`, w.Body)
}

func TestListPostsSelectTagsError(t *testing.T) {
	db, mock, err := sqlxmock.Newx()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT (id, title, body) FROM post")).WithArgs(0, handlers.MaxPostsLimit).WillReturnRows(
		sqlxmock.NewRows([]string{
			"id", "title", "body",
		}).AddRow(
			1, "title", "body",
		).AddRow(
			2, "title", "body",
		),
	)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT (name, post_id) FROM tag WHERE post_id IN")).WillReturnError(fmt.Errorf("test db error"))

	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()

	sut := handlers.ListPostsJSON(db)
	sut(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assertBodyEqual(t, `{"error":"test db error"}`, w.Body)
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
