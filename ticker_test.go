/*
   Copyright 2020 Joseph Cumines

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package behaviortree

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewTicker_panic1(t *testing.T) {
	defer func() {
		r := recover()
		if s := fmt.Sprint(r); r == nil || s != "behaviortree.NewTicker nil context" {
			t.Fatal("unexpected panic", s)
		}
	}()
	NewTicker(nil, 1, func() (Tick, []Node) {
		return nil, nil
	})
	t.Error("expected a panic")
}

func TestNewTicker_panic2(t *testing.T) {
	defer func() {
		r := recover()
		if s := fmt.Sprint(r); r == nil || s != "behaviortree.NewTicker duration <= 0" {
			t.Fatal("unexpected panic", s)
		}
	}()
	NewTicker(context.Background(), 0, func() (Tick, []Node) {
		return nil, nil
	})
	t.Error("expected a panic")
}

func TestNewTicker_panic3(t *testing.T) {
	defer func() {
		r := recover()
		if s := fmt.Sprint(r); r == nil || s != "behaviortree.NewTicker nil node" {
			t.Fatal("unexpected panic", s)
		}
	}()
	NewTicker(context.Background(), 1, nil)
	t.Error("expected a panic")
}

func TestNewTicker_run(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 50)

		endGoroutines := runtime.NumGoroutine()

		if endGoroutines > startGoroutines {
			t.Error("started with", startGoroutines, "goroutines and ended with", endGoroutines)
		}
	}()

	var (
		mutex   sync.Mutex
		counter int
	)

	node := func() (Tick, []Node) {
		return func(children []Node) (Status, error) {
			mutex.Lock()
			defer mutex.Unlock()
			counter++
			time.Sleep(time.Millisecond)
			return Success, nil
		}, nil
	}

	type stopComplain int

	c := NewTicker(
		context.WithValue(context.Background(), stopComplain(1), 2),
		time.Millisecond*5,
		node,
	)

	time.Sleep(time.Millisecond * 50)

	v := c.(*tickerCore)

	if v.node == nil || reflect.ValueOf(node).Pointer() != reflect.ValueOf(v.node).Pointer() {
		//noinspection GoPrintFunctions
		t.Error("unexpected node", v.node)
	}

	if v.ctx == nil || v.ctx.Value(stopComplain(1)) != 2 {
		t.Error("unexpected context", v.ctx)
	}

	if v.cancel == nil {
		t.Error("nil cancel")
	}

	if v.err != nil {
		t.Error("unexpected error", v.err)
	}

	if v.stop == nil || v.done == nil {
		t.Error("nil chans")
	}

	if v.ticker == nil {
		t.Error("nil ticker")
	}

	func() {
		mutex.Lock()
		defer mutex.Unlock()
		if counter < 0 {
			t.Error("bad counter", counter)
		}
	}()

	if err := c.Err(); err != nil {
		t.Error("unexpected error call", err)
	}

	if d := c.Done(); d == nil || d != v.done {
		t.Error("unexpected done call", d)
	}

	func() {
		start := time.Now()

		c.Stop()

		<-c.Done()

		diff := time.Now().Sub(start)

		if diff > time.Millisecond*20 {
			t.Fatal("unexpected diff", diff)
		}

		if err := c.Err(); err != nil {
			t.Fatal(err)
		}
	}()
}

func TestNewTicker_runError(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 50)

		endGoroutines := runtime.NumGoroutine()

		if endGoroutines > startGoroutines {
			t.Error("started with", startGoroutines, "goroutines and ended with", endGoroutines)
		}
	}()

	expected := errors.New("some_error")

	node := func() (Tick, []Node) {
		return func(children []Node) (Status, error) {
			return 0, expected
		}, nil
	}

	startedAt := time.Now()
	defer func() {
		diff := time.Now().Sub(startedAt)
		if diff > time.Millisecond*20 {
			t.Error("unexpected diff", diff)
		}
	}()

	c := NewTicker(
		context.Background(),
		time.Millisecond,
		node,
	)

	<-c.Done()

	if err := c.Err(); err != expected {
		t.Error("unexpected error", err)
	}
}

func TestNewTicker_runCancel(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 50)

		endGoroutines := runtime.NumGoroutine()

		if endGoroutines > startGoroutines {
			t.Error("started with", startGoroutines, "goroutines and ended with", endGoroutines)
		}
	}()

	node := func() (Tick, []Node) {
		return func(children []Node) (Status, error) {
			time.Sleep(time.Millisecond)
			return Success, nil
		}, nil
	}

	startedAt := time.Now()
	defer func() {
		diff := time.Now().Sub(startedAt)
		if diff > time.Millisecond*20 {
			t.Error("unexpected diff", diff)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*5)
	defer cancel()

	c := NewTicker(
		ctx,
		time.Millisecond,
		node,
	)

	<-c.Done()

	if err := c.Err(); err == nil || err.Error() != "context deadline exceeded" {
		t.Error("unexpected error", err)
	}
}

func TestNewTickerStopOnFailure_success(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 100)
		endGoroutines := runtime.NumGoroutine()
		if startGoroutines < endGoroutines {
			t.Errorf("ended with %d more goroutines", endGoroutines-startGoroutines)
		}
	}()
	var (
		mutex  sync.Mutex
		count  int
		ticker = NewTickerStopOnFailure(
			context.Background(),
			time.Millisecond*50,
			func() (Tick, []Node) {
				return func(children []Node) (Status, error) {
					mutex.Lock()
					defer mutex.Unlock()
					if len(children) != 5 {
						t.Error("bad children", len(children))
					}
					count++
					if count == 5 {
						return Failure, nil
					}
					return Success, nil
				}, make([]Node, 5)
			},
		)
	)
	defer ticker.Stop()
	timer := time.NewTimer(time.Millisecond * 350)
	defer timer.Stop()
	startedAt := time.Now()
	select {
	case <-timer.C:
		t.Fatal("expected done")
	case <-ticker.Done():
	}
	duration := time.Now().Sub(startedAt)
	if duration < time.Millisecond*170 {
		t.Error(duration.String())
	}
	mutex.Lock()
	defer mutex.Unlock()
	if err := ticker.Err(); err != nil {
		t.Error(err)
	}
}

func TestNewTickerStopOnFailure_error(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 100)
		endGoroutines := runtime.NumGoroutine()
		if startGoroutines < endGoroutines {
			t.Errorf("ended with %d more goroutines", endGoroutines-startGoroutines)
		}
	}()
	ticker := NewTickerStopOnFailure(
		context.Background(),
		time.Millisecond*50,
		func() (Tick, []Node) {
			return func(children []Node) (Status, error) {
				return Failure, errors.New("some_error")
			}, make([]Node, 5)
		},
	)
	defer ticker.Stop()
	<-ticker.Done()
	if ticker.Err() == nil {
		t.Fatal("expected an error")
	}
}

func TestNewTickerStopOnFailure_nilNode(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 100)
		endGoroutines := runtime.NumGoroutine()
		if startGoroutines < endGoroutines {
			t.Errorf("ended with %d more goroutines", endGoroutines-startGoroutines)
		}
	}()
	defer func() {
		if r := fmt.Sprint(recover()); r != "behaviortree.NewTickerStopOnFailure nil node" {
			t.Error(r)
		}
	}()
	NewTickerStopOnFailure(nil, 0, nil)
}

func TestNewTickerStopOnFailure_nilTick(t *testing.T) {
	startGoroutines := runtime.NumGoroutine()
	defer func() {
		time.Sleep(time.Millisecond * 100)
		endGoroutines := runtime.NumGoroutine()
		if startGoroutines < endGoroutines {
			t.Errorf("ended with %d more goroutines", endGoroutines-startGoroutines)
		}
	}()
	ticker := NewTickerStopOnFailure(
		context.Background(),
		time.Millisecond*10,
		func() (tick Tick, nodes []Node) {
			return
		},
	)
	<-ticker.Done()
	if err := ticker.Err(); err == nil || err.Error() != "behaviortree.Node cannot tick a node with a nil tick" {
		t.Error(err)
	}
}
