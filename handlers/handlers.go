package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"
)

const (
	insertPost = `INSERT INTO post (title, body) values ($1, $2);`
	insertTags = `INSERT INTO tag (name, post_id) values (:name, :post_id);`
)

type Post struct {
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

func addPost(db *sqlx.DB, post Post) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Insert post.
	res, err := tx.Exec(insertPost, post.Title, post.Body)
	if err != nil {
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
	_, err = db.NamedExec(insertTags, tags)
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
