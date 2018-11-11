package jsonstore

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Value interface {
	Clone() Value
}

type Store interface {
	Load() (x Value)
	Store(x Value)
	Save() error
	Close()
	Context() context.Context
}

type jsonStore struct {
	mu    *sync.Mutex
	ctx   context.Context
	store *atomic.Value
	fn    string
}

func (js *jsonStore) open(data Value) error {
	f, err := os.Open(js.fn)
	if err != nil {
		if os.IsNotExist(err) {
			js.store.Store(data)
			return nil
		}
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	err = dec.Decode(data)
	if err != nil {
		return err
	}
	js.store.Store(data)
	return nil
}

func (js *jsonStore) Load() (x Value) {
	return js.store.Load().(Value)
}

func (js *jsonStore) Store(x Value) {
	js.mu.Lock()
	defer js.mu.Unlock()
	// copy on write
	js.store.Store(x.Clone())
}

func (js *jsonStore) Save() error {
	f, err := os.Create(js.fn)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	return enc.Encode(js.Load())
}

func (js *jsonStore) Close() {
	// nothing
}

func (js *jsonStore) Context() context.Context {
	return js.ctx
}

type storeCancel struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     *sync.WaitGroup
	once   *sync.Once
	store  Store
}

func NewStoreCancel(store Store) *storeCancel {
	ictx, cancel := context.WithCancel(store.Context())
	return &storeCancel{
		ctx:    ictx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
		once:   &sync.Once{},
		store:  store,
	}
}

// data must contain initial value for store it in json file (if not exists)
func NewJsonStoreCancel(ctx context.Context, fn string, data Value, ticksave *time.Ticker) (*storeCancel, error) {
	js := &jsonStore{
		mu:    &sync.Mutex{},
		ctx:   ctx,
		store: &atomic.Value{},
		fn:    fn,
	}
	err := js.open(data)
	if err != nil {
		return nil, err
	}
	sc := NewStoreCancel(js)
	if ticksave != nil {
		sc.Go(func(dn <-chan struct{}, st Store) {
			for {
				select {
				case <-dn:
					return
				case <-ticksave.C:
					st.Save()
				}
			}
		})
	}
	return sc, nil
}

func (sc *storeCancel) Close() {
	sc.once.Do(func() {
		sc.cancel()
		sc.wg.Wait()
		sc.store.Save()
		sc.store.Close()
	})
}

func (sc *storeCancel) Context() context.Context {
	return sc.ctx
}

func (sc *storeCancel) Done() <-chan struct{} {
	return sc.ctx.Done()
}

func (sc *storeCancel) Store() Store {
	return sc.store
}

func (sc *storeCancel) Go(f func(done <-chan struct{}, store Store)) {
	sc.wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup, store Store) {
		defer wg.Done()
		f(ctx.Done(), store)
	}(sc.ctx, sc.wg, sc.store)
}
