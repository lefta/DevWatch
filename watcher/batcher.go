// Stolen from https://github.com/gohugoio/hugo/blob/master/watcher/batcher.go
// Here is the original copyright:
//
// Copyright 2015 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package watcher

import (
	"time"

	fsnotify "gopkg.in/fsnotify.v1"
)

type batcher struct {
	watcher *fsnotify.Watcher
	done    chan struct{}

	events chan []fsnotify.Event // Events are returned on this channel
}

func newBatcher() (*batcher, error) {
	var err error
	b := &batcher{}

	b.watcher, err = fsnotify.NewWatcher()
	b.done = make(chan struct{}, 1)
	b.events = make(chan []fsnotify.Event, 1)

	if err == nil {
		go b.run()
	}

	return b, err
}

func (b *batcher) run() {
	tick := time.Tick(50 * time.Millisecond)
	evs := make([]fsnotify.Event, 0)
OuterLoop:
	for {
		select {
		case ev := <-b.watcher.Events:
			evs = append(evs, ev)
		case <-tick:
			if len(evs) == 0 {
				continue
			}
			b.events <- evs
			evs = make([]fsnotify.Event, 0)
		case <-b.done:
			break OuterLoop
		}
	}
	close(b.done)
}

func (b *batcher) close() {
	b.done <- struct{}{}
	b.watcher.Close()
}
