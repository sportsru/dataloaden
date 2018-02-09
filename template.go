package main

import "text/template"

var tpl = template.Must(template.New("generated").Parse(`
// generated by github.com/vektah/dataloaden ; DO NOT EDIT

package {{.package}}
        
import (
    "sync"
    "time"
    
    {{if .import}}"{{.import}}"{{end}}
)
        
type {{.Name}}Loader struct {
	// this method provides the data for the loader
	fetch func(keys []string) ([]*{{.type}}, []error)

	// how long to done before sending a batch
	wait time.Duration

	// this will limit the maximum number of keys to send in one batch, 0 = no limit
	maxBatch int

	// INTERNAL

	// lazily created cache
	cache map[string]*{{.type}}

	// the current batch. keys will continue to be collected until timeout is hit,
	// then everything will be sent to the fetch method and out to the listeners
	batch *{{.name}}Batch

	// mutex to prevent races
	mu sync.Mutex
}

type {{.name}}Batch struct {
	keys    []string
	data    []*{{.type}}
	error   []error
	closing bool
	done    chan struct{}
	timer   time.Timer
}

// Load a {{.Name}} by key, batching and caching will be applied automatically
func (l *{{.Name}}Loader) Load(key string) (*{{.type}}, error) {
	return l.LoadThunk(key)()
}

// LoadThunk returns a function that when called will block waiting for a {{.Name}}.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *{{.Name}}Loader) LoadThunk(key string) func() (*{{.type}}, error) {
	l.mu.Lock()
	if it, ok := l.cache[key]; ok {
		l.mu.Unlock()
		return func() (*{{.type}}, error) {
			return it, nil
		}
	}
	if l.batch == nil {
		l.batch = &{{.name}}Batch{done: make(chan struct{})}
	}
	batch := l.batch
	pos := batch.keyIndex(l, key)
	l.mu.Unlock()

	return func() (*{{.type}}, error) {
		<-batch.done

		if batch.error[pos] == nil {
			l.mu.Lock()
			if l.cache == nil {
				l.cache = map[string]*{{.type}}{}
			}
			l.cache[key] = batch.data[pos]
			l.mu.Unlock()
		}

		return batch.data[pos], batch.error[pos]
	}
}

// LoadAll fetches many keys at once. It will be broken into appropriate sized
// sub batches depending on how the loader is configured
func (l *{{.Name}}Loader) LoadAll(keys []string) ([]*{{.type}}, []error) {
	results := make([]func() (*{{.type}}, error), len(keys))

	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}

	{{.name}}s := make([]*{{.type}}, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range results {
		{{.name}}s[i], errors[i] = thunk()
	}
	return {{.name}}s, errors
}

// keyIndex will return the location of the key in the batch, if its not found
// it will add the key to the batch
func (b *{{.name}}Batch) keyIndex(l *{{.Name}}Loader, key string) int {
	for i, existingKey := range b.keys {
		if key == existingKey {
			return i
		}
	}

	pos := len(b.keys)
	b.keys = append(b.keys, key)
	if pos == 0 {
		go b.startTimer(l)
	}

	if l.maxBatch != 0 && pos >= l.maxBatch-1 {
		if !b.closing {
			b.closing = true
			l.batch = nil
			go b.end(l)
		}
	}

	return pos
}

func (b *{{.name}}Batch) startTimer(l *{{.Name}}Loader) {
	time.Sleep(l.wait)
	l.mu.Lock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		l.mu.Unlock()
		return
	}

	l.batch = nil
	l.mu.Unlock()

	b.end(l)
}

func (b *{{.name}}Batch) end(l *{{.Name}}Loader) {
	b.data, b.error = l.fetch(b.keys)
	close(b.done)
}
`))