package core

type JobDispatcher interface {
	Dispatch(job ProcessJob)
}
