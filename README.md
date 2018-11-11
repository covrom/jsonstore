# jsonstore
Json store with atomic.Value copy on write and context closing

Usage:

```go
type Config struct{
    A,B string
}

func (cf *Config) Clone() jsonstore.Value{
    return &Config{
        A: cf.A,
        B: cf.B,
    }
}

tmpfn := "/tmp/testJsonStoreCancel.json"

str := &Config{"aaa", "bbb"}
store, err := NewJsonStoreCancel(context.Background(), "/tmp/testJsonStoreCancel.json", str, time.NewTicker(time.Second), false)
if err != nil {
    fmt.Printf("NewJsonStoreCancel() error = %v", err)
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
    fmt.Println(time.Now().String())
    store.Close()
})

<-time.After(3 * time.Second)

b, err := ioutil.ReadFile(tmpfn)
if err != nil {
    fmt.Printf("NewJsonStoreCancel() error = %v", err)
    return
}
fmt.Println(string(b))

```