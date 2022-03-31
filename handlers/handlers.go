package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jmoiron/sqlx"
)

const (
	MaxPostsLimit = 1000
)

const (
	insertPostQuery    = `INSERT INTO post (title, body) values ($1, $2) RETURNING id;`
	insertTagsQuery    = `INSERT INTO tag (name, post_id) values (:name, :post_id);`
	listPostsQuery     = `SELECT * FROM post OFFSET $1 LIMIT $2;`
	listPostsTagsQuery = `SELECT * FROM tag WHERE post_id IN (?);`
)

type Post struct {
	ID    int      `db:"id"`
	Title string   `db:"title" json:"title"`
	Body  string   `db:"body" json:"body"`
	Tags  []string `db:"tags" json:"tags"`
}

type Tag struct {
	ID     int    `db:"id"`
	Name   string `db:"name"`
	PostID int    `db:"post_id"`
}

type Err struct {
	Msg string `json:"error"`
}

func AddPostJSON(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		post := Post{}
		err := json.NewDecoder(r.Body).Decode(&post)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		err = addPost(db, post)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
		}
	}
}

func addPost(db *sqlx.DB, post Post) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("error while begin transaction: %s", err.Error())
	}

	// Insert post.
	postID := 0
	err = tx.QueryRow(insertPostQuery, post.Title, post.Body).Scan(&postID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error while insert post into db: %s", err.Error())
	}

	// Build slice of tags structs for batch insert via NamedExec.
	tags := make([]Tag, len(post.Tags))
	for i := range tags {
		tags[i].Name = post.Tags[i]
		tags[i].PostID = postID
	}

	// Insert tags associated with post.
	_, err = tx.NamedExec(insertTagsQuery, tags)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error while insert post tags into db: %s", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error while commit transaction: %s", err.Error())
	}

	return nil
}

func ListPostsJSON(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offset, limit, err := getListPostsPaginationURLParams(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest)
			return
		}

		posts, err := listPosts(db, offset, limit)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError)
			return
		}

		postsJSON, err := json.Marshal(posts)
		if err != nil {
			handleError(w, fmt.Errorf("error while marshal list of posts: %s", err), http.StatusInternalServerError)
			return
		}
		w.Write(postsJSON)
	}
}

func listPosts(db *sqlx.DB, offset, limit int) ([]Post, error) {
	// Select posts with offset and limit.
	posts := []Post{}
	err := db.Select(&posts, listPostsQuery, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("error while fetch list of posts from db: %s", err.Error())
	}

	// Get list of posts IDs.
	postsIDs := []int{}
	for _, post := range posts {
		postsIDs = append(postsIDs, post.ID)
	}

	// Select posts tags.
	query, args, err := sqlx.In(listPostsTagsQuery, postsIDs)
	if err != nil {
		return nil, fmt.Errorf("error while construct query fetch list of posts from db: %s", err.Error())
	}
	query = db.Rebind(query)

	tags := []Tag{}
	err = db.Select(&tags, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while fetch list of posts tags from db: %s", err.Error())
	}

	// Create map of posts tags.
	postsTags := make(map[int][]string, len(posts))
	for _, post := range posts {
		postsTags[post.ID] = []string{}
	}
	for _, tag := range tags {
		postsTags[tag.PostID] = append(postsTags[tag.PostID], tag.Name)
	}

	// Arrange tags to their posts.
	for i, post := range posts {
		posts[i].Tags = postsTags[post.ID]
	}

	return posts, nil
}

func getListPostsPaginationURLParams(r *http.Request) (offset, limit int, err error) {
	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")
	if offsetStr == "" {
		offset = 0
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, fmt.Errorf("incorrect offset URL parameter, actual: %s, expected: positive integer", offsetStr)
		}
	}
	if limitStr == "" {
		limit = MaxPostsLimit
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, fmt.Errorf("incorrect limit URL parameter, actual: %s, expected: positive integer", limitStr)
		}
	}

	if offset < 0 {
		return 0, 0, fmt.Errorf("offset cannot be less than 0")
	}
	if limit < 0 {
		return 0, 0, fmt.Errorf("limit cannot be less than 0")
	}
	if limit > MaxPostsLimit {
		return 0, 0, fmt.Errorf("limit cannot be greater than %d", MaxPostsLimit)
	}

	return
}

func handleError(w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	errStruct := Err{Msg: err.Error()}
	errMsg, err := json.Marshal(errStruct)
	if err != nil {
		fmt.Fprintf(w, "error while marshal error: %s", err.Error())
		return
	}
	w.Write(errMsg)
}
