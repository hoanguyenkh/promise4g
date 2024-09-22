package promise4g

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Promise represents a computation that will eventually be completed with a value of type T or an error.
type Promise[T any] struct {
	value atomic.Value
	err   atomic.Value
	done  chan struct{}
	once  sync.Once
}

// New creates a new Promise with the given task
func New[T any](task func(resolve func(T), reject func(error))) *Promise[T] {
	return NewWithPool(task, defaultPool)
}

// NewWithPool creates a new Promise with the given task and pool
func NewWithPool[T any](task func(resolve func(T), reject func(error)), pool Pool) *Promise[T] {
	if task == nil {
		panic("task must not be nil")
	}
	if pool == nil {
		panic("pool must not be nil")
	}
	p := &Promise[T]{
		done: make(chan struct{}),
	}
	pool.Go(func() {
		defer p.handlePanic()
		task(p.resolve, p.reject)
	})
	return p
}

// Await waits for the Promise to be resolved or rejected
func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var t T
		return t, ctx.Err()
	case <-p.done:
		if err := p.err.Load(); err != nil {
			var t T
			return t, err.(error)
		}
		return p.value.Load().(T), nil
	}
}

func (p *Promise[T]) resolve(value T) {
	p.once.Do(func() {
		p.value.Store(value)
		close(p.done)
	})
}

func (p *Promise[T]) reject(err error) {
	p.once.Do(func() {
		p.err.Store(err)
		close(p.done)
	})
}

func (p *Promise[T]) handlePanic() {
	if r := recover(); r != nil {
		var err error
		switch v := r.(type) {
		case error:
			err = v
		default:
			err = fmt.Errorf("%v", v)
		}
		p.reject(err)
	}
}

// All waits for all promises to be resolved
func All[T any](ctx context.Context, promises ...*Promise[T]) *Promise[[]T] {
	return AllWithPool(ctx, defaultPool, promises...)
}

// AllWithPool waits for all promises to be resolved using the given pool
func AllWithPool[T any](ctx context.Context, pool Pool, promises ...*Promise[T]) *Promise[[]T] {
	if len(promises) == 0 {
		panic("missing promises")
	}

	return NewWithPool(func(resolve func([]T), reject func(error)) {
		results := make([]T, len(promises))
		var wg sync.WaitGroup
		wg.Add(len(promises))

		for i, p := range promises {
			i, p := i, p
			pool.Go(func() {
				defer wg.Done()
				result, err := p.Await(ctx)
				if err != nil {
					reject(err)
					return
				}
				results[i] = result
			})
		}

		wg.Wait()
		resolve(results)
	}, pool)
}

// Race returns a promise that resolves or rejects as soon as one of the promises resolves or rejects
func Race[T any](ctx context.Context, promises ...*Promise[T]) *Promise[T] {
	return NewWithPool(func(resolve func(T), reject func(error)) {
		for _, p := range promises {
			go func(p *Promise[T]) {
				result, err := p.Await(ctx)
				if err != nil {
					reject(err)
				} else {
					resolve(result)
				}
			}(p)
		}
	}, defaultPool)
}

// Then chains a new Promise to the current one
func Then[A, B any](p *Promise[A], ctx context.Context, resolve func(A) (B, error)) *Promise[B] {
	return ThenWithPool(p, ctx, resolve, defaultPool)
}

// ThenWithPool chains a new Promise to the current one using the given pool
func ThenWithPool[A, B any](p *Promise[A], ctx context.Context, resolve func(A) (B, error), pool Pool) *Promise[B] {
	return NewWithPool(func(resolveB func(B), reject func(error)) {
		result, err := p.Await(ctx)
		if err != nil {
			reject(err)
			return
		}

		resultB, err := resolve(result)
		if err != nil {
			reject(err)
			return
		}

		resolveB(resultB)
	}, pool)
}

// Catch handles errors in the Promise chain
func Catch[T any](p *Promise[T], ctx context.Context, reject func(error) error) *Promise[T] {
	return CatchWithPool(p, ctx, reject, defaultPool)
}

// CatchWithPool handles errors in the Promise chain using the given pool
func CatchWithPool[T any](p *Promise[T], ctx context.Context, reject func(error) error, pool Pool) *Promise[T] {
	return NewWithPool(func(resolve func(T), internalReject func(error)) {
		result, err := p.Await(ctx)
		if err != nil {
			internalReject(reject(err))
		} else {
			resolve(result)
		}
	}, pool)
}

// Finally executes a function regardless of whether the promise is fulfilled or rejected
func Finally[T any](p *Promise[T], ctx context.Context, fn func()) *Promise[T] {
	return NewWithPool(func(resolve func(T), reject func(error)) {
		result, err := p.Await(ctx)
		fn()
		if err != nil {
			reject(err)
		} else {
			resolve(result)
		}
	}, defaultPool)
}

// Timeout returns a new Promise that rejects if the original Promise doesn't resolve within the specified duration
func Timeout[T any](p *Promise[T], d time.Duration) *Promise[T] {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	return NewWithPool(func(resolve func(T), reject func(error)) {
		defer cancel()
		result, err := p.Await(ctx)
		if err != nil {
			reject(err)
		} else {
			resolve(result)
		}
	}, defaultPool)
}
