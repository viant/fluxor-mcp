package context

import (
	"context"

	"github.com/viant/mcp/client"
)

type clientKey string

var ClientKey = clientKey("client")

func WithClient(ctx context.Context, client client.Interface) context.Context {
	return context.WithValue(ctx, ClientKey, client)
}

func Client(ctx context.Context) (client.Interface, bool) {
	ret := ctx.Value(ClientKey)
	if ret == nil {
		return nil, false
	}
	aClient, ok := ret.(client.Interface)
	return aClient, ok
}
