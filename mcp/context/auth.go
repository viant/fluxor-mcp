package context

import (
	"context"

	"github.com/viant/mcp/client/auth/transport"
)

func WithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, transport.ContextAuthTokenKey, token)
}

func AuthToken(ctx context.Context) (string, bool) {
	ret := ctx.Value(transport.ContextAuthTokenKey)
	if ret == nil {
		return "", false
	}
	aClient, ok := ret.(string)
	return aClient, ok
}
