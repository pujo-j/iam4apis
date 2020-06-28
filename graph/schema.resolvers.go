package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"time"

	"github.com/pujo-j/iam4apis/graph/generated"
	"github.com/pujo-j/iam4apis/graph/model"
)

func (r *adminEventResolver) Admin(ctx context.Context, obj *model.AdminEvent) (*model.User, error) {
	return r.Db.GetUser(ctx, obj.AdminID)
}

func (r *adminEventResolver) User(ctx context.Context, obj *model.AdminEvent) (*model.User, error) {
	return r.Db.GetUser(ctx, obj.UserID)
}

func (r *mutationResolver) EditUser(ctx context.Context, u model.EditUser) (*model.User, error) {
	return r.Db.UpdateUser(ctx, u)
}

func (r *queryResolver) Users(ctx context.Context, from *string) ([]*model.User, error) {
	return r.Db.GetUsers(ctx, from)
}

func (r *queryResolver) SearchUsers(ctx context.Context, namePart string) ([]*model.User, error) {
	return r.Db.SearchUsers(ctx, namePart)
}

func (r *queryResolver) AdminEvents(ctx context.Context, from *time.Time) ([]*model.AdminEvent, error) {
	return r.Db.GetEvents(ctx, from)
}

func (r *queryResolver) UserInRole(ctx context.Context, user string, path string, role string) (*bool, error) {
	full_user, err := r.Db.GetUser(ctx, user)
	if err != nil {
		return nil, err
	}
	isInRole := full_user.IsInRole(role, path)
	return &isInRole, nil
}

// AdminEvent returns generated.AdminEventResolver implementation.
func (r *Resolver) AdminEvent() generated.AdminEventResolver { return &adminEventResolver{r} }

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type adminEventResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
