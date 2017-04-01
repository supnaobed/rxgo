package rxgo

import "sync"

type Connect struct {
	In  Observable
	Out Observable
}

type Observable chan interface{}
type Obsrver chan Observable

type Connector chan Observable
type Emitter chan Observable

type Connector1 struct {
	InOut *chan Observable
	Start *chan interface{}
}

func (o Observable) fromTo(from, to int) Observable {
	go func() {
		for i := from; i < to; i++ {
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
			go f(o)
			out <- o
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
		for i := from; i < to; i++ {
			o <- i
		}
		close(o)
	}
	in.Start(out, fromTo)
	return out
}

func Range(from, to int) *Connector1 {
	c := Connector1{}
	s := make(chan interface{})
	c.Start = &s
	io := make(chan Observable)
	c.InOut = &io
	fromTo := func(o Observable) {
		for i := from; i < to; i++ {
			o <- i
		}
		close(o)
	}
	go func() {
		for {
			<-s
			o := make(Observable)
			io <- o
			go fromTo(o)
		}
	}()
	return &c
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

func (c *Connector1) Map(f func(interface{}) interface{}) *Connector1 {
	in := c.InOut
	out := make(chan Observable)
	c.InOut = &out
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
	go func() {
		for {
			o := <-*in
			go func() {
				out <- work(o)
			}()
		}
	}()
	return c
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

func (in Connector) MergeWith(cs ...Connector) Connector {
	var wg sync.WaitGroup
	out := make(Connector)
	cs = append(cs, in)

	output := func(c Connector) {
		for {
			o := <-c
			out <- o
		}
		wg.Done()
	}
	wg.Add(len(cs))

	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func (c1 *Connector1) MergeWith(cs ...*Connector1) *Connector1 {
	var wg sync.WaitGroup
	in := *c1.InOut
	out := make(chan Observable)
	c1.InOut = &out
	cs = append(cs, c1)

	c1StartIn := *c1.Start
	s := make(chan interface{})
	c1.Start = &s

	output := func(c *Connector1) {
		if c == c1 {
			c1StartIn <- true
			for {
				o := <-in
				out <- o
			}
		} else {
			*c.Start <- true
			for {
				o := <-*c.InOut
				out <- o
			}
		}
		wg.Done()
	}
	wg.Add(len(cs))

	go func() {
		<-s
		for _, c := range cs {
			go output(c)
		}
	}()

	go func() {
		wg.Wait()
		close(out)
	}()
	return c1
}

func (in Connector) FlatMap(f func(interface{}) Connector) Connector {
	out := make(Connector)
	var wgGlobal sync.WaitGroup
	var wgLocal sync.WaitGroup

	output := func(o Connector) {
		wgLocal.Add(1)
		for n := range o {
			out <- n
		}
		wgLocal.Done()
	}

	wgGlobal.Add(1)
	go func() {
		for {
			i, ok := <-in
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
				if ok {
					f(i)
				} else {
					break
				}
			}
		}
	}()
}

func (c *Connector1) Subscribe(f func(interface{})) {
	go func() {
		for {
			o := <-*c.InOut
			for {
				i, ok := <-o
				if ok {
					f(i)
				} else {
					break
				}
			}
		}
	}()
	*c.Start <- true
}
