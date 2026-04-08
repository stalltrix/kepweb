package postdb

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type Reply struct {
	ID       int
	User     string
	Meta     string
	Me       bool
	Post     string
	Time     int64
	Tag      uint16
	Hex      string
	MetaTime int64
}

type Post struct {
	PostHex  string
	TagID    uint16
	Owner    string
	Replies  []Reply
	LastTime int64
	TypeID   byte
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=536870912",
	}

	for _, p := range pragmas {
		db.Exec(p)
	}

	schema := `
CREATE TABLE IF NOT EXISTS post (
	post_hex TEXT PRIMARY KEY,
	tag_id INTEGER,
	owner TEXT,
	last_time INTEGER,
	type_id INTEGER
);

CREATE TABLE IF NOT EXISTS reply (
	id INTEGER,
	post_hex TEXT,
	user TEXT,
	meta TEXT,
	me INTEGER,
	post TEXT,
	post_time INTEGER,
	tag INTEGER,
	hex TEXT,
	meta_time INTEGER
);

CREATE INDEX IF NOT EXISTS idx_reply_post_time
ON reply(post_hex, post_time);
`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func (s *Store) Store(p *Post) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
INSERT OR REPLACE INTO post(post_hex, tag_id, owner, last_time, type_id)
VALUES(?,?,?,?,?)`,
		p.PostHex, p.TagID, p.Owner, p.LastTime, p.TypeID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM reply WHERE post_hex=?`, p.PostHex)
	if err != nil {
		tx.Rollback()
		return err
	}

	stmt, err := tx.Prepare(`
INSERT INTO reply(id, post_hex, user, meta, me, post, post_time, tag, hex, meta_time)
VALUES(?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, r := range p.Replies {
		me := 0
		if r.Me {
			me = 1
		}

		_, err = stmt.Exec(
			r.ID,
			p.PostHex,
			r.User,
			r.Meta,
			me,
			r.Post,
			r.Time,
			r.Tag,
			r.Hex,
			r.MetaTime,
		)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) Load(postHex string) (*Post, error) {
	row := s.db.QueryRow(`
SELECT post_hex, tag_id, owner, last_time, type_id
FROM post WHERE post_hex=?`, postHex)

	p := &Post{}
	err := row.Scan(&p.PostHex, &p.TagID, &p.Owner, &p.LastTime, &p.TypeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
SELECT id, user, meta, me, post, post_time, tag, hex, meta_time
FROM reply
WHERE post_hex=?
ORDER BY post_time`, postHex)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r Reply
		var me int

		err = rows.Scan(
			&r.ID,
			&r.User,
			&r.Meta,
			&me,
			&r.Post,
			&r.Time,
			&r.Tag,
			&r.Hex,
			&r.MetaTime,
		)
		if err != nil {
			return nil, err
		}

		if me == 1 {
			r.Me = true
		}

		p.Replies = append(p.Replies, r)
	}

	return p, nil
}

func (s *Store) Delete(postHex string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM reply WHERE post_hex=?`, postHex)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec(`DELETE FROM post WHERE post_hex=?`, postHex)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}