package core

import "context"

type JobDispatcher interface {
	Dispatch(ctx context.Context, job ProcessJob)
}
