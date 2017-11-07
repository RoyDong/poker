package utils

import (
    "sync"
)


/*
Function used to bind to specified event
when the event triggered it will be executed
*/
type EventHandler func(args ...interface{})

type IEvent interface {
    AddHandler(name string, handler EventHandler)
    AddSyncHandler(name string, handler EventHandler)
    ClearHandlers(name string)
    ClearAllHandlers()
    Trigger(name string, args ...interface{})
}

/*
Event manages events and handlers
an event is just a collection of functions
*/
type Event struct {
    //异步协程执行
    events map[string][]EventHandler

    //同步执行
    syncEvents map[string][]EventHandler

    mu sync.RWMutex
}

/*
AddHandler adds an EventHandler to an event by name
if there is no event with the name, it will be created automatically
*/
func (e *Event) AddHandler(name string, handler EventHandler) {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.events == nil {
        e.events = make(map[string][]EventHandler)
    }
    handlers, has := e.events[name]
    if !has {
        handlers = make([]EventHandler, 0, 1)
    }
    e.events[name] = append(handlers, handler)
}

/*
AddHandler adds an EventHandler to an event by name
if there is no event with the name, it will be created automatically
*/
func (e *Event) AddSyncHandler(name string, handler EventHandler) {
    e.mu.Lock()
    defer e.mu.Unlock()
    if e.syncEvents == nil {
        e.syncEvents = make(map[string][]EventHandler)
    }
    handlers, has := e.syncEvents[name]
    if !has {
        handlers = make([]EventHandler, 0, 1)
    }
    e.syncEvents[name] = append(handlers, handler)
}

/*
ClearHandlers removes all handlers binds to event by name
*/
func (e *Event) ClearHandlers(name string) {
    e.mu.Lock()
    defer e.mu.Unlock()
    delete(e.events, name)
    delete(e.syncEvents, name)
}

/*
ClearHandlers removes all handlers in e
*/
func (e *Event) ClearAllHandlers() {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.events = make(map[string][]EventHandler)
    e.syncEvents = make(map[string][]EventHandler)
}

/*
Trigger triggers event by name, sync handlers are executed in FIFO order
*/
func (e *Event) Trigger(name string, args ...interface{}) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    if handlers, has := e.events[name]; has && len(handlers) > 0 {
        for _, handler := range handlers {
            go handler(args...)
        }
    }
    if handlers, has := e.syncEvents[name]; has && len(handlers) > 0 {
        for _, handler := range handlers {
            handler(args...)
        }
    }
}


