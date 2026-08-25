package auth

import (
	"context"

	"time3api/app/user"
)

const userContextKey int = iota

func withUser(ctx context.Context, usr *user.User) context.Context {
	return context.WithValue(ctx, userContextKey, usr)
}

func UserFromContext(ctx context.Context) (*user.User, bool) {
	usr, ok := ctx.Value(userContextKey).(*user.User)
	return usr, ok
}
