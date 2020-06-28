package main

import (
	_ "github.com/99designs/gqlgen/graphql/handler"
	_ "github.com/99designs/gqlgen/graphql/introspection"
	_ "github.com/99designs/gqlgen/graphql/playground"
	_ "github.com/dgraph-io/ristretto"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/vektah/gqlparser/v2"
)
