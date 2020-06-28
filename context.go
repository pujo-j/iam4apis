package main

import (
	"context"
	"github.com/pujo-j/iam4apis/graph/model"
)

type contextKey struct {
	name string
}

var userKey = &contextKey{"user"}

func ForContext(ctx context.Context) *model.User {
	raw, _ := ctx.Value(userKey).(*model.User)
	return raw
}
