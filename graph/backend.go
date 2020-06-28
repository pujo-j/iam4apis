package graph

import (
	"context"
	"github.com/pujo-j/iam4apis/graph/model"
	"net/http"
	"time"
)

type Db interface {
	Login(ctx context.Context) (*model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUsers(ctx context.Context, fromId *string) ([]*model.User, error)
	SearchUsers(ctx context.Context, namePart string) ([]*model.User, error)
	UpdateUser(ctx context.Context, newUser model.EditUser) (*model.User, error)
	GetEvents(ctx context.Context, from *time.Time) ([]*model.AdminEvent, error)
	EnrichUser(ctx context.Context, email string, fullName string, profile string) (*model.User, error)
	MiddleWare(r *http.Request) context.Context
}
