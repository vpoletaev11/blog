package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
)

const (
	insertPostQuery    = `INSERT INTO post (title, body) values (?, ?);`
	insertTagsQuery    = `INSERT INTO tag (name, post_id) values (:name, :post_id);`
	listPostsQuery     = `SELECT (id, title, body) FROM post OFFSET ? LIMIT ?;`
	listPostsTagsQuery = `SELECT (name, post_id) FROM tag WHERE post_id IN (?);`
)

type Post struct {
	ID    int      `db:"id"`
	Title string   `db:"title" json:"title"`
	Body  string   `db:"body" json:"body"`
	Tags  []string `db:"tags" json:"tags"`
}

type Tag struct {
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
			w.WriteHeader(http.StatusBadRequest)
			errStruct := Err{Msg: "incorrect request data"}
			errMsg, err := json.Marshal(errStruct)
			// TODO: return text error
			if err != nil {
				panic(err)
			}
			w.Write(errMsg)
			return
		}

		err = addPost(db, post)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errStruct := Err{Msg: err.Error()}
			errMsg, err := json.Marshal(errStruct)
			if err != nil {
				panic(err)
			}
			w.Write(errMsg)
		}
	}
}

func ListPostsJSON(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		offsetStr := r.URL.Query().Get("offset")
		limitStr := r.URL.Query().Get("limit")
		offset := 0
		limit := 0
		if offsetStr == "" {
			offset = 0
		}
		if limitStr == "" {
			limit = 100
		}

		// TODO: add offset and limit URL parameters validation

		posts, err := listPosts(db, offset, limit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errStruct := Err{Msg: err.Error()}
			errMsg, err := json.Marshal(errStruct)
			if err != nil {
				panic(err)
			}
			w.Write(errMsg)
			return
		}

		postsJSON, err := json.Marshal(posts)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errStruct := Err{Msg: err.Error()}
			errMsg, err := json.Marshal(errStruct)
			if err != nil {
				panic(err)
			}
			w.Write(errMsg)
			return
		}
		w.Write(postsJSON)
	}
}

func addPost(db *sqlx.DB, post Post) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Insert post.
	res, err := tx.Exec(insertPostQuery, post.Title, post.Body)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Get id of inserted post.
	postID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// Build slice of tags structs for batch insert via NamedExec.
	tags := make([]Tag, len(post.Tags))
	for i := range tags {
		tags[i].Name = post.Tags[i]
		tags[i].PostID = int(postID)
	}

	// Insert tags associated with post.
	_, err = db.NamedExec(insertTagsQuery, tags)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func listPosts(db *sqlx.DB, offset, limit int) ([]Post, error) {
	// Select posts with offset and limit.
	posts := []Post{}
	err := db.Select(&posts, listPostsQuery, offset, limit)
	if err != nil {
		return nil, err
	}

	// Get list of posts IDs.
	postsIDs := []int{}
	for _, post := range posts {
		postsIDs = append(postsIDs, post.ID)
	}

	// Select posts tags.
	query, args, err := sqlx.In(listPostsTagsQuery, postsIDs)
	if err != nil {
		return nil, err
	}
	query = db.Rebind(query)

	tags := []Tag{}
	err = db.Select(&tags, query, args...)
	if err != nil {
		return nil, err
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
