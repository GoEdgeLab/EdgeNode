// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

import (
	"context"
	"database/sql"
)

type Stmt struct {
	db      *DB
	rawStmt *sql.Stmt
	query   string

	enableStat bool
}

func NewStmt(db *DB, rawStmt *sql.Stmt, query string) *Stmt {
	return &Stmt{
		db:      db,
		rawStmt: rawStmt,
		query:   query,
	}
}

func (this *Stmt) EnableStat() {
	this.enableStat = true
}

func (this *Stmt) ExecContext(ctx context.Context, args ...interface{}) (sql.Result, error) {
	// check database status
	if this.db.BeginUpdating() {
		defer this.db.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.ExecContext(ctx, args...)
}

func (this *Stmt) Exec(args ...interface{}) (sql.Result, error) {
	// check database status
	if this.db.BeginUpdating() {
		defer this.db.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.Exec(args...)
}

func (this *Stmt) QueryContext(ctx context.Context, args ...interface{}) (*sql.Rows, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.QueryContext(ctx, args...)
}

func (this *Stmt) Query(args ...interface{}) (*sql.Rows, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.Query(args...)
}

func (this *Stmt) QueryRowContext(ctx context.Context, args ...interface{}) *sql.Row {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.QueryRowContext(ctx, args...)
}

func (this *Stmt) QueryRow(args ...interface{}) *sql.Row {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.QueryRow(args...)
}

func (this *Stmt) Close() error {
	return this.rawStmt.Close()
}
