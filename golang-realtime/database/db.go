package database

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTimeout = 3 * time.Second

func New(connStr string) (*pgx.Conn, error) {
	log.Println("connStr:", connStr)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		panic(err)
	}

	return conn, nil
}

func NewPool(connStr string) (*pgxpool.Pool, error) {
	log.Println("connStr:", connStr)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	conn, err := pgxpool.New(ctx, connStr)
	if err != nil {
		panic(err)
	}

	return conn, nil
}
