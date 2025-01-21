package controllers

import (
	"github.com/jackc/pgx/v5"

	"github.com/ponty96/simple-web-app/internal/db"
)

type Controller struct {
	db      *pgx.Conn
	queries *db.Queries
}

func NewController(d *pgx.Conn) *Controller {
	client := db.New(d)
	return &Controller{
		db:      d,
		queries: client,
	}
}
