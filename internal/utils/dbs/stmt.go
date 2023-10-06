// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.

package dbs

import (
	"context"
	"database/sql"
	fsutils "github.com/TeaOSLab/EdgeNode/internal/utils/fs"
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

func (this *Stmt) ExecContext(ctx context.Context, args ...any) (result sql.Result, err error) {
	// check database status
	if this.db.BeginUpdating() {
		defer this.db.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	fsutils.WriteBegin()
	result, err = this.rawStmt.ExecContext(ctx, args...)
	fsutils.WriteEnd()
	return
}

func (this *Stmt) Exec(args ...any) (result sql.Result, err error) {
	// check database status
	if this.db.BeginUpdating() {
		defer this.db.EndUpdating()
	} else {
		return nil, errDBIsClosed
	}

	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}

	fsutils.WriteBegin()
	result, err = this.rawStmt.Exec(args...)
	fsutils.WriteEnd()
	return
}

func (this *Stmt) QueryContext(ctx context.Context, args ...any) (*sql.Rows, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.QueryContext(ctx, args...)
}

func (this *Stmt) Query(args ...any) (*sql.Rows, error) {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	rows, err := this.rawStmt.Query(args...)
	if err != nil {
		return nil, err
	}
	var rowsErr = rows.Err()
	if rowsErr != nil {
		_ = rows.Close()
		return nil, rowsErr
	}
	return rows, nil
}

func (this *Stmt) QueryRowContext(ctx context.Context, args ...any) *sql.Row {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.QueryRowContext(ctx, args...)
}

func (this *Stmt) QueryRow(args ...any) *sql.Row {
	if this.enableStat {
		defer SharedQueryStatManager.AddQuery(this.query).End()
	}
	return this.rawStmt.QueryRow(args...)
}

func (this *Stmt) Close() error {
	return this.rawStmt.Close()
}
