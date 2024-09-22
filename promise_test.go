package promise4g

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	conc "github.com/sourcegraph/conc/pool"

	"github.com/stretchr/testify/require"
)

func TestPromise_One(t *testing.T) {
	t.Run("Happy", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(string), reject func(error)) {
			resolve("one")
		})
		result, err := p.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, "one", result)
	})

	t.Run("Reject", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("error"))
		})
		_, err := p.Await(ctx)
		require.Error(t, err)
	})

	t.Run("Panic", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(string), reject func(error)) {
			panic(errors.New("panic"))
		})
		_, err := p.Await(ctx)
		fmt.Println(err)
		require.Error(t, err)
	})

	t.Run("MultipleResolves", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(string), reject func(error)) {
			resolve("one")
			resolve("two") // This should be ignored
		})
		result, err := p.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, "one", result)
	})

	t.Run("MultipleRejects", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("first error"))
			reject(errors.New("second error")) // This should be ignored
		})
		_, err := p.Await(ctx)
		require.Error(t, err)
		require.Equal(t, "first error", err.Error())
	})
}

func TestPromise_All(t *testing.T) {
	t.Run("AllHappy", func(t *testing.T) {
		ctx := context.Background()
		p1 := New(func(resolve func(string), reject func(error)) {
			resolve("one")
		})

		p2 := New(func(resolve func(string), reject func(error)) {
			resolve("two")
		})

		p3 := New(func(resolve func(string), reject func(error)) {
			resolve("five")
		})
		p := All(ctx, p1, p2, p3)
		results, err := p.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, []string{"one", "two", "five"}, results)
	})

	t.Run("AllContainReject", func(t *testing.T) {
		ctx := context.Background()
		p1 := New(func(resolve func(string), reject func(error)) {
			resolve("one")
		})

		p2 := New(func(resolve func(string), reject func(error)) {
			resolve("two")
		})

		p3 := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("error"))
		})
		p := All(ctx, p1, p2, p3)
		result, err := p.Await(ctx)
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("AllWithCanceledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		p1 := New(func(resolve func(int), reject func(error)) {
			time.Sleep(100 * time.Millisecond)
			resolve(1)
		})

		p2 := New(func(resolve func(int), reject func(error)) {
			time.Sleep(200 * time.Millisecond)
			resolve(2)
		})
		// Cancel the context before promises complete
		go func() {
			time.Sleep(150 * time.Millisecond)
			cancel()
		}()

		allPromise := All(ctx, p1, p2)
		result, err := allPromise.Await(context.Background())
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("AllMixedResolveReject", func(t *testing.T) {
		ctx := context.Background()
		p1 := New(func(resolve func(string), reject func(error)) {
			resolve("one")
		})

		p2 := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("error"))
		})

		p := All(ctx, p1, p2)
		result, err := p.Await(ctx)
		require.Error(t, err)
		require.Nil(t, result)
	})

	t.Run("AllWithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		p1 := New(func(resolve func(int), reject func(error)) {
			time.Sleep(100 * time.Millisecond)
			resolve(1)
		})

		p2 := New(func(resolve func(int), reject func(error)) {
			time.Sleep(200 * time.Millisecond)
			resolve(2)
		})

		allPromise := All(ctx, p1, p2)
		result, err := allPromise.Await(context.Background())
		require.Error(t, err)
		require.Nil(t, result)
	})
}

func TestPromise_Then(t *testing.T) {
	t.Run("ThenSuccess", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(int), reject func(error)) {
			resolve(1)
		})

		thenPromise := Then(p, ctx, func(val int) (string, error) {
			return fmt.Sprintf("Value is %d", val), nil
		})

		result, err := thenPromise.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, "Value is 1", result)
	})

	t.Run("ThenFailure", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(int), reject func(error)) {
			reject(errors.New("initial error"))
		})

		thenPromise := Then(p, ctx, func(val int) (string, error) {
			return "", errors.New("should not reach here")
		})

		result, err := thenPromise.Await(ctx)
		require.Error(t, err)
		require.Empty(t, result)
	})

	t.Run("ThenSuccessButThenPromiseError", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(int), reject func(error)) {
			resolve(1)
		})

		thenPromise := Then(p, ctx, func(val int) (string, error) {
			return "", errors.New("then promise error")
		})

		result, err := thenPromise.Await(ctx)
		require.Error(t, err)
		require.Equal(t, "then promise error", err.Error())
		require.Empty(t, result)
	})
}

func TestPromise_Catch(t *testing.T) {
	t.Run("CatchNoError", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(int), reject func(error)) {
			resolve(1)
		})

		catchPromise := Catch(p, ctx, func(err error) error {
			return errors.New("should not reach here")
		})

		result, err := catchPromise.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, result)
	})

	t.Run("CatchSuccess", func(t *testing.T) {
		ctx := context.Background()
		p := New(func(resolve func(int), reject func(error)) {
			reject(errors.New("initial error"))
		})

		catchPromise := Catch(p, ctx, func(err error) error {
			return errors.New("handled error")
		})

		result, err := catchPromise.Await(ctx)
		require.Error(t, err)
		require.Equal(t, "handled error", err.Error())
		require.Empty(t, result)
	})
}

func TestNewWithPool(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		pool Pool
	}{
		{
			name: "default",
			pool: newDefaultPool(),
		},
		{
			name: "conc",
			pool: func() Pool {
				return FromConcPool(conc.New())
			}(),
		},
		{
			name: "ants",
			pool: func() Pool {
				antsPool, err := ants.NewPool(0)
				require.NoError(t, err)
				return FromAntsPool(antsPool)
			}(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := NewWithPool(func(resolve func(string), reject func(error)) {
				resolve(test.name)
			}, test.pool)

			val, err := p.Await(ctx)
			require.NoError(t, err)
			require.NotNil(t, val)
			require.Equal(t, test.name, val)
		})
	}
}

func TestCheckAllConcurrent(t *testing.T) {

	ctx := context.Background()
	start := time.Now()

	p1 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(100 * time.Millisecond)
		resolve("one")
	})

	p2 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(200 * time.Millisecond)
		resolve("two")
	})

	p3 := New(func(resolve func(string), reject func(error)) {
		time.Sleep(300 * time.Millisecond)
		resolve("three")
	})

	p := All(ctx, p1, p2, p3)
	results, err := p.Await(ctx)
	elapsed := time.Since(start)
	fmt.Println(elapsed)

	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, results)
	require.Less(t, elapsed, 350*time.Millisecond, "Promises did not run concurrently")
}

func BenchmarkNewWithPool(b *testing.B) {
	ctx := context.Background()

	tests := []struct {
		name string
		pool Pool
	}{
		{
			name: "default",
			pool: newDefaultPool(),
		},
		{
			name: "conc",
			pool: func() Pool {
				return FromConcPool(conc.New())
			}(),
		},
		{
			name: "ants",
			pool: func() Pool {
				antsPool, err := ants.NewPool(0)
				require.NoError(b, err)
				return FromAntsPool(antsPool)
			}(),
		},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				p := NewWithPool(func(resolve func(string), reject func(error)) {
					resolve(test.name)
				}, test.pool)

				val, err := p.Await(ctx)
				require.NoError(b, err)
				require.NotNil(b, val)
				require.Equal(b, test.name, val)
			}
		})
	}
}

func TestPromise_Race(t *testing.T) {
	t.Run("RaceWithFastestResolve", func(t *testing.T) {
		ctx := context.Background()
		p1 := New(func(resolve func(string), reject func(error)) {
			time.Sleep(100 * time.Millisecond)
			resolve("slow")
		})
		p2 := New(func(resolve func(string), reject func(error)) {
			resolve("fast")
		})
		p3 := New(func(resolve func(string), reject func(error)) {
			time.Sleep(50 * time.Millisecond)
			resolve("medium")
		})

		racePromise := Race(ctx, p1, p2, p3)
		result, err := racePromise.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, "fast", result)
	})

	t.Run("RaceWithFastestReject", func(t *testing.T) {
		ctx := context.Background()
		p1 := New(func(resolve func(string), reject func(error)) {
			time.Sleep(100 * time.Millisecond)
			resolve("slow")
		})
		p2 := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("fast error"))
		})
		p3 := New(func(resolve func(string), reject func(error)) {
			time.Sleep(50 * time.Millisecond)
			resolve("medium")
		})

		racePromise := Race(ctx, p1, p2, p3)
		_, err := racePromise.Await(ctx)
		require.Error(t, err)
		require.Equal(t, "fast error", err.Error())
	})
}

func TestPromise_Finally(t *testing.T) {
	t.Run("FinallyAfterResolve", func(t *testing.T) {
		ctx := context.Background()
		finallyExecuted := false
		p := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		finalPromise := Finally(p, ctx, func() {
			finallyExecuted = true
		})

		result, err := finalPromise.Await(ctx)
		require.NoError(t, err)
		require.Equal(t, "success", result)
		require.True(t, finallyExecuted)
	})

	t.Run("FinallyAfterReject", func(t *testing.T) {
		ctx := context.Background()
		finallyExecuted := false
		p := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("error"))
		})

		finalPromise := Finally(p, ctx, func() {
			finallyExecuted = true
		})

		_, err := finalPromise.Await(ctx)
		require.Error(t, err)
		require.True(t, finallyExecuted)
	})
}

func TestPromise_Timeout(t *testing.T) {
	t.Run("TimeoutBeforeResolve", func(t *testing.T) {
		p := New(func(resolve func(string), reject func(error)) {
			time.Sleep(200 * time.Millisecond)
			resolve("too late")
		})

		timeoutPromise := Timeout(p, 100*time.Millisecond)
		_, err := timeoutPromise.Await(context.Background())
		require.Error(t, err)
	})

	t.Run("ResolveBeforeTimeout", func(t *testing.T) {
		p := New(func(resolve func(string), reject func(error)) {
			time.Sleep(50 * time.Millisecond)
			resolve("on time")
		})

		timeoutPromise := Timeout(p, 100*time.Millisecond)
		result, err := timeoutPromise.Await(context.Background())
		require.NoError(t, err)
		require.Equal(t, "on time", result)
	})
}
