package pl

// event is been pushed into the queue for execution and it is not been executed
// at once but deferred its execution in event queue.
type EventQueue interface {
	// invoked when a event is emitted, the first argument is the event name and
	// the second is its corresponding context
	OnEvent(string, Val) error

	// Drain the event queue
	Drain(*Evaluator, *Module, func(string, error) bool) int

	// Size of the queue that is pending
	PendingSize() int

	// Clear all the pending event internally
	Clear() int
}

type eventEntry struct {
	name    string
	context Val
}

type defEventQueue struct {
	q []eventEntry
}

func (d *defEventQueue) OnEvent(n string, v Val) error {
	d.q = append(d.q, eventEntry{
		name:    n,
		context: v,
	})
	return nil
}

func (d *defEventQueue) Drain(ev *Evaluator,
	p *Module,
	onError func(string, error) bool,
) int {
	count := 0

	for len(d.q) != 0 {
		sz := len(d.q)
		last := d.q[sz-1]
		d.q = d.q[:sz-1]
		count++

		_, err := ev.EvalDeferred(last.name, last.context, p)

		if !onError(
			last.name,
			err,
		) {
			break
		}
	}
	return count
}

func (d *defEventQueue) PendingSize() int {
	return len(d.q)
}

func (d *defEventQueue) Clear() int {
	x := len(d.q)
	d.q = []eventEntry{}
	return x
}
