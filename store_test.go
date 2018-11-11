package jsonstore

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync/atomic"
	"testing"
	"time"
)

type Config struct {
	A, B string
}

func (cf *Config) Clone() Value {
	return &Config{
		A: cf.A,
		B: cf.B,
	}
}
func TestNewJsonStoreCancel(t *testing.T) {

	tmpfn := "/tmp/testJsonStoreCancel.json"

	str := &Config{"aaa", "bbb"}
	store, err := NewJsonStoreCancel(context.Background(), "/tmp/testJsonStoreCancel.json", str, time.NewTicker(time.Second), false)
	if err != nil {
		t.Errorf("NewJsonStoreCancel() error = %v", err)
		return
	}

	cnt := int64(0)

	f := func(d <-chan struct{}, st Store) {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-d:
				return
			case t := <-tick.C:
				c := atomic.AddInt64(&cnt, 1)
				set := st.Load().(*Config)
				set.A = t.String()
				set.B = fmt.Sprintf("changed by iteration %d", c)
				st.Store(set)
			}
		}
	}

	store.Go(f)
	store.Go(f)
	store.Go(f)

	time.AfterFunc(2*time.Second, func() {
		t.Log(time.Now().String())
		store.Close()
	})

	<-time.After(3 * time.Second)
	b, err := ioutil.ReadFile(tmpfn)
	if err != nil {
		t.Errorf("NewJsonStoreCancel() error = %v", err)
		return
	}
	fmt.Println(string(b))

}
