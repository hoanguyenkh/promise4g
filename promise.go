package promise4g

import (
	"context"
	"fmt"
	"sync"
)

// Promise represents a computation that will eventually be completed with a value of type T or an error.
type Promise[T any] struct {
	ctx   context.Context
	value T
	err   error
	ch    chan struct{}
	once  sync.Once
}

func New[T any](
	ctx context.Context,
	task func(resolve func(T), reject func(error))) *Promise[T] {
	return NewWithPool(ctx, task, defaultPool)
}

func NewWithPool[T any](
	ctx context.Context,
	task func(resolve func(T), reject func(error)),
	pool Pool) *Promise[T] {
	if task == nil {
		panic("task must not be nil")
	}
	if pool == nil {
		panic("pool must not be nil")
	}
	var t T
	p := &Promise[T]{
		ctx:   ctx,
		value: t,
		err:   nil,
		ch:    make(chan struct{}),
		once:  sync.Once{},
	}
	pool.Go(func() {
		defer p.handlePanic()
		task(p.resolve, p.reject)
	})
	return p
}

func (p *Promise[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-ctx.Done():
		var t T
		return t, ctx.Err()
	case <-p.ch:
		return p.value, p.err
	}
}

func (p *Promise[T]) resolve(value T) {
	p.once.Do(func() {
		p.value = value
		close(p.ch)
	})
}

func (p *Promise[T]) reject(err error) {
	p.once.Do(func() {
		p.err = err
		close(p.ch)
	})
}

func (p *Promise[T]) handlePanic() {
	err := recover()
	if err == nil {
		return
	}
	switch v := err.(type) {
	case error:
		p.reject(v)
	default:
		p.reject(fmt.Errorf("%+v", v))
	}
}

func All[T any](
	ctx context.Context,
	promises ...*Promise[T],
) *Promise[[]T] {
	return AllWithPool(ctx, defaultPool, promises...)
}

func AllWithPool[T any](
	ctx context.Context,
	pool Pool,
	promises ...*Promise[T],
) *Promise[[]T] {
	if len(promises) == 0 {
		panic("missing promises")
	}

	return NewWithPool(ctx, func(resolve func([]T), reject func(error)) {
		resultsChan := make(chan tuple[T, int], len(promises))
		errsChan := make(chan error, len(promises))

		for idx, p := range promises {
			idx := idx
			_ = ThenWithPool(p, ctx, func(data T) (T, error) {
				resultsChan <- tuple[T, int]{_1: data, _2: idx}
				return data, nil
			}, pool)
			_ = CatchWithPool(p, ctx, func(err error) error {
				errsChan <- err
				return err
			}, pool)
		}

		results := make([]T, len(promises))
		for idx := 0; idx < len(promises); idx++ {
			select {
			case result := <-resultsChan:
				results[result._2] = result._1
			case err := <-errsChan:
				reject(err)
				return
			}
		}
		resolve(results)
	}, pool)
}

func Then[A, B any](
	p *Promise[A],
	ctx context.Context,
	resolve func(A) (B, error),
) *Promise[B] {
	return ThenWithPool(p, ctx, resolve, defaultPool)
}

func ThenWithPool[A, B any](
	p *Promise[A],
	ctx context.Context,
	resolve func(A) (B, error),
	pool Pool,
) *Promise[B] {
	return NewWithPool(ctx, func(resolveB func(B), reject func(error)) {
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

func Catch[T any](
	p *Promise[T],
	ctx context.Context,
	reject func(err error) error,
) *Promise[T] {
	return CatchWithPool(p, ctx, reject, defaultPool)
}

func CatchWithPool[T any](
	p *Promise[T],
	ctx context.Context,
	reject func(err error) error,
	pool Pool,
) *Promise[T] {
	return NewWithPool(ctx, func(resolve func(T), internalReject func(error)) {
		result, err := p.Await(ctx)
		if err != nil {
			internalReject(reject(err))
		} else {
			resolve(result)
		}
	}, pool)
}

type tuple[T1, T2 any] struct {
	_1 T1
	_2 T2
}
