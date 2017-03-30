package rxgo

import "sync"

type Observable chan interface{}

func (o Observable) fromTo(from, to int) Observable {
	c := o
	go func() {
		for i := from; i <= to; i++ {
			c <- i
		}
		close(c)
	}()
	return c
}

func FromTo(from, to int) Observable {
	o := make(Observable)
	o.fromTo(from, to)
	return o
}

func (o Observable) Map(f func(interface{}) interface{}) Observable {
	out := make(Observable)
	go func() {
		for {
			i, ok := <-o
			if !ok {
				close(out)
				return
			}
			out <- f(i)
		}
	}()
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

func (o Observable) ToSlice() Observable {
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

func (o Observable) Subscribe(f func(interface{})) {
	go func() {
		for {
			i, ok := <-o
			if !ok {
				return
			}
			f(i)
		}
	}()
}
