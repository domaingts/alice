package cache

import (
	"container/list"
	"sync"
)

// Lru simple, fast lru cache implementation
type Lru interface {
	Get(key any) (value any, ok bool)
	GetKeyFromValue(value any) (key any, ok bool)
	PeekKeyFromValue(value any) (key any, ok bool) // Peek means check but NOT bring to top
	Put(key, value any)
}

type lru struct {
	capacity         int
	doubleLinkedlist *list.List
	keyToElement     *sync.Map
	valueToElement   *sync.Map
	mu               *sync.Mutex
}

type lruElement struct {
	key   any
	value any
}

// NewLru initializes a lru cache
func NewLru(cap int) Lru {
	return &lru{
		capacity:         cap,
		doubleLinkedlist: list.New(),
		keyToElement:     new(sync.Map),
		valueToElement:   new(sync.Map),
		mu:               new(sync.Mutex),
	}
}

func (l *lru) Get(key any) (value any, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if v, ok := l.keyToElement.Load(key); ok {
		element := v.(*list.Element)
		l.doubleLinkedlist.MoveToFront(element)
		return element.Value.(*lruElement).value, true
	}
	return nil, false
}

func (l *lru) GetKeyFromValue(value any) (key any, ok bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if k, ok := l.valueToElement.Load(value); ok {
		element := k.(*list.Element)
		l.doubleLinkedlist.MoveToFront(element)
		return element.Value.(*lruElement).key, true
	}
	return nil, false
}

func (l *lru) PeekKeyFromValue(value any) (key any, ok bool) {
	if k, ok := l.valueToElement.Load(value); ok {
		element := k.(*list.Element)
		return element.Value.(*lruElement).key, true
	}
	return nil, false
}

func (l *lru) Put(key, value any) {
	l.mu.Lock()
	e := &lruElement{key, value}
	if v, ok := l.keyToElement.Load(key); ok {
		element := v.(*list.Element)
		element.Value = e
		l.doubleLinkedlist.MoveToFront(element)
	} else {
		element := l.doubleLinkedlist.PushFront(e)
		l.keyToElement.Store(key, element)
		l.valueToElement.Store(value, element)
		if l.doubleLinkedlist.Len() > l.capacity {
			toBeRemove := l.doubleLinkedlist.Back()
			l.doubleLinkedlist.Remove(toBeRemove)
			l.keyToElement.Delete(toBeRemove.Value.(*lruElement).key)
			l.valueToElement.Delete(toBeRemove.Value.(*lruElement).value)
		}
	}
	l.mu.Unlock()
}
