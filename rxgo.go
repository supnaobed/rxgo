package rxgo

import "sync"

type Connect struct {
}

type Observable chan interface{}
type Obsrver chan Observable

type Connector chan Observable
type Emitter chan Observable

func (o Observable) fromTo(from, to int) Observable {
	go func() {
		for i := from; i <= to; i++ {
			o <- i
		}
		close(o)
	}()
	return o
}

func (in Connector) Start(out Connector, f func(Observable)) {
	go func() {
		for {
			o := <-in
			out <- o
			go f(o)
		}
	}()
}

func (in Connector) Between(out Connector, f func(Observable) Observable) {
	go func() {
		for {
			o := <-in
			go func() { out <- f(o) }()
		}
	}()
}

func (in Connector) Range(from, to int) Connector {
	out := make(Connector)
	fromTo := func(o Observable) {
		for i := from; i <= to; i++ {
			o <- i
		}
		close(o)
	}
	in.Start(out, fromTo)
	return out
}

func (in Connector) Map(f func(interface{}) interface{}) Connector {
	out := make(Connector)
	work := func(o Observable) Observable {
		out1 := make(Observable)
		go func() {
			for {
				i, ok := <-o
				if !ok {
					close(out1)
					break
				}
				out1 <- f(i)
			}
		}()
		return out1
	}
	in.Between(out, work)
	return out
}

func (o Observable) Filter(f func(interface{}) bool) Observable {
	out := make(Observable)
	go func() {
		for {
			i, ok := <-o
			if !ok {
				close(out)
				return
			}
			if f(i) {
				out <- i
			}
		}
	}()
	return out
}

func (in Connector) ToSlice() Connector {
	out := make(Connector)

	f := func(o Observable) Observable {
		out := make(Observable)
		go func() {
			t := make([]interface{}, 0)
			for {
				i, ok := <-o
				if ok {
					t = append(t, i)
				} else {
					out <- t
					close(out)
					return
				}
			}
		}()
		return out
	}

	in.Between(out, f)

	return out
}

func (o Observable) MergeWith(os ...Observable) Observable {
	var wg sync.WaitGroup
	out := make(Observable)
	os = append(os, o)
	output := func(o Observable) {
		for n := range o {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(os))

	for _, o := range os {
		go output(o)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func (o Observable) FlatMap(f func(interface{}) Observable) Observable {
	out := make(Observable)
	var wgGlobal sync.WaitGroup
	var wgLocal sync.WaitGroup

	output := func(o Observable) {
		wgLocal.Add(1)
		for n := range o {
			out <- n
		}
		wgLocal.Done()
	}

	wgGlobal.Add(1)
	go func() {
		for {
			i, ok := <-o
			if !ok {
				wgGlobal.Done()
				return
			}
			go output(f(i))
		}
	}()
	go func() {
		wgGlobal.Wait()
		wgLocal.Wait()
		close(out)
	}()
	return out
}

type Source struct {
	Next     func(interface{})
	Complete func()
}

func Create(f func(*Source)) Observable {
	out := make(Observable)
	s := Source{}
	s.Next = func(i interface{}) {
		out <- i
	}
	s.Complete = func() {
		close(out)
	}
	go f(&s)
	return out
}

func Use(s *Source) Observable {
	out := make(Observable)
	s.Next = func(i interface{}) {
		out <- i
	}
	s.Complete = func() {
		close(out)
	}
	return out
}

func (out Connector) Subscribe(f func(interface{})) {
	go func() {
		for {
			o := <-out
			for {
				i, ok := <-o
				if !ok {
					return
				}
				f(i)
			}
		}
	}()
}
