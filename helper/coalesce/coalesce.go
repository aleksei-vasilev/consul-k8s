package coalesce

import (
	"context"
	"time"
)

// Coalesce will wait until the given function blocks for at least `quiet`
// time or is returning for at least `max` time. This enables easy and safe
// data coalescing.
//
// `quiet` is the time waited between `f` returns, and is reset on each return.
// If quiet is reached, then Coalesce returns and `f` is not called anymore.
// `max` is the maximum time to wait for `f` to return. This does not reset.
//
// The callback f must be well-behaved with the context passed in. The context
// will be cancelled prior to Coalesce returning, and Coalesce will block until
// the function call returns. This ensures that there are no data races once
// Coalesce returns.
//
// If the given ctx is cancelled, this function also cancels. It follows the
// same behavior as if the timeout were reached.
//
// Real world example: imagine you have a function processing input data,
// and you'd like to accumulate as much input data as possible to batch process
// it. The logic you'd say is: keep receiving data until I don't receive any
// within Q time or at most M time passes. Q is usually much shorter than M.
// This means if the data input is "quiet" enough, continue, otherwise wait
// until some maximum amount of time and still continue. This is what this
// function does generally.
func Coalesce(ctx context.Context, quiet, max time.Duration, f func(context.Context)) {
	// Setup a max duration timeout
	ctx, maxCloser := context.WithTimeout(ctx, max)
	defer maxCloser()

	// Setup the variables to store the context and cancellation function
	// for each iteration. We do this outside of the loop so that we can
	// setup a defer to ensure we cleanup any resources associated with the
	// context.
	var curCtx context.Context
	var curCancel context.CancelFunc = func() {}
	defer func() {
		// Defer has to be wrapped in a func() so we don't dereferene the
		// wrong curCancel. The function call is deferred but curCancel is
		// dereferenced immediately.
		curCancel()
	}()

	for {
		// Cancel the previous context
		curCancel()

		// Create a context with our quiet period
		curCtx, curCancel = context.WithTimeout(ctx, quiet)

		// Call the function
		f(curCtx)

		// If the context ended, then we're also done. If the context didn't
		// end, then the function processed successfully and we continue.
		err := curCtx.Err()
		if err != nil {
			return
		}
	}
}
