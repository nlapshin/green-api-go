package greenapi

import "context"

type contextKeyOp struct{}

func withOperation(ctx context.Context, op string) context.Context {
	return context.WithValue(ctx, contextKeyOp{}, op)
}

func operationFromContext(ctx context.Context) string {
	s, _ := ctx.Value(contextKeyOp{}).(string)
	return s
}
