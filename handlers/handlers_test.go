package handlers_test

import (
	"blog/handlers"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
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

	assert.Equal(t, 500, w.Code)
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

	assert.Equal(t, 500, w.Code)
	assertBodyEqual(t, `{"error":"test db error"}`, w.Body)
}

func assertBodyEqual(t *testing.T, expected string, actual *bytes.Buffer) {
	bodyBytes, err := ioutil.ReadAll(actual)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	assert.Equal(t, expected, bodyString)
}
