# go-behaviortree

Package behaviortree provides a functional style implementation of behavior trees
in Golang with no fluff.

Go doc: [https://godoc.org/github.com/joeycumines/go-behaviortree](https://godoc.org/github.com/joeycumines/go-behaviortree)

Wikipedia: [Behavior tree - AI, robotics, and control](https://en.wikipedia.org/wiki/Behavior_tree_(artificial_intelligence,_robotics_and_control))

```go
type (
	// Node represents an node in a tree, that can be ticked
	Node func() (Tick, []Node)

	// Tick represents the logic for a node, which may or may not be stateful
	Tick func(children []Node) (Status, error)
)

// Tick runs the node's tick function with it's children
func (n Node) Tick() (Status, error)
```

- Core implementation as above
- Sequence and Selector also provided as per the
    [Wikipedia page](https://en.wikipedia.org/wiki/Behavior_tree_(artificial_intelligence,_robotics_and_control))
- Async and Sync wrappers allow for the definition of time consuming logic that gets performed 
    in serial, but without blocking the tick operation.

## Example Usage

The example below is straight from `example_test.go`.

TODO: more complicated example using `Async` and `Sync`

```go
func ExampleSimpleTicker() {
	var (
		counter                    = 0
		nodeGuardCounterLessThan10 = NewNode(
			func(children []Node) (Status, error) {
				if counter < 10 {
					return Success, nil
				}
				return Failure, nil
			},
			nil,
		)
		nodeGuardCounterLessThan20 = NewNode(
			func(children []Node) (Status, error) {
				if counter < 20 {
					return Success, nil
				}
				return Failure, nil
			},
			nil,
		)
		newNodePrintCounter = func(name string) Node {
			return NewNode(
				func(children []Node) (Status, error) {
					fmt.Printf("%s: %d\n", name, counter)
					return Success, nil
				},
				nil,
			)
		}
		nodeIncrementCounter = NewNode(
			func(children []Node) (Status, error) {
				counter++
				return Success, nil
			},
			nil,
		)
		nodeRoot = NewNode(
			Selector,
			[]Node{
				NewNode(
					Sequence,
					[]Node{
						nodeGuardCounterLessThan10,
						nodeIncrementCounter,
						newNodePrintCounter("< 10"),
					},
				),
				NewNode(
					Sequence,
					[]Node{
						nodeGuardCounterLessThan20,
						nodeIncrementCounter,
						newNodePrintCounter("< 20"),
					},
				),
			},
		)
		tickerRoot = NewTickerStopOnFailure(context.Background(), time.Millisecond, nodeRoot)
	)
	<-tickerRoot.Done()
	if err := tickerRoot.Err(); err != nil {
		panic(err)
	}
	// Output:
	// < 10: 1
	// < 10: 2
	// < 10: 3
	// < 10: 4
	// < 10: 5
	// < 10: 6
	// < 10: 7
	// < 10: 8
	// < 10: 9
	// < 10: 10
	// < 20: 11
	// < 20: 12
	// < 20: 13
	// < 20: 14
	// < 20: 15
	// < 20: 16
	// < 20: 17
	// < 20: 18
	// < 20: 19
	// < 20: 20
}
```
