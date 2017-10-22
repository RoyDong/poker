package lib

import "sync"


/*
Function used to bind to specified event
when the event triggered it will be executed
*/
type EventHandler func(args ...interface{})

/*
Event manages events and handlers
an event is just a collection of functions
*/
type Event struct {
    //异步协程执行
    events map[string][]EventHandler

    //同步执行
    syncEvents map[string][]EventHandler

    locker sync.Mutex
}

/*
NewEvent creates and initialize an Event struct
*/
func NewEvent() *Event {
    return &Event{
        make(map[string][]EventHandler),
        make(map[string][]EventHandler),
        sync.Mutex{},
    }
}

/*
AddHandler adds an EventHandler to an event by name
if there is no event with the name, it will be created automatically
*/
func (e *Event) AddHandler(name string, handler EventHandler) int {
    e.locker.Lock()
    defer e.locker.Unlock()
    handlers, has := e.events[name]
    if !has {
        handlers = make([]EventHandler, 0, 1)
    }
    e.events[name] = append(handlers, handler)
    return len(e.events[name]) - 1
}

/*
AddHandler adds an EventHandler to an event by name
if there is no event with the name, it will be created automatically
*/
func (e *Event) AddSyncHandler(name string, handler EventHandler) int {
    e.locker.Lock()
    defer e.locker.Unlock()
    handlers, has := e.syncEvents[name]
    if !has {
        handlers = make([]EventHandler, 0, 1)
    }
    e.syncEvents[name] = append(handlers, handler)
    return len(e.syncEvents[name]) - 1
}

/*
RemoveHandler removes handler binds to event by name
*/
func (e *Event) RemoveHandler(name string, id int) {
    e.locker.Lock()
    defer e.locker.Unlock()
    if handlers, has := e.events[name]; has && len(handlers) > 0 {
        e.events[name] = append(handlers[:id], handlers[id+1:]...)
    }
}

/*
RemoveHandler removes handler binds to event by name
*/
func (e *Event) RemoveSyncHandler(name string, id int) {
    e.locker.Lock()
    defer e.locker.Unlock()
    if handlers, has := e.syncEvents[name]; has && len(handlers) > 0 {
        e.syncEvents[name] = append(handlers[:id], handlers[id+1:]...)
    }
}

/*
ClearHandlers removes all handlers binds to event by name
*/
func (e *Event) ClearHandlers(name string) {
    delete(e.events, name)
    delete(e.syncEvents, name)
}

/*
ClearHandlers removes all handlers in e
*/
func (e *Event) ClearAllHandlers() {
    e.events = make(map[string][]EventHandler)
    e.syncEvents = make(map[string][]EventHandler)
}

/*
Trigger triggers event by name, sync handlers are executed in FIFO order
*/
func (e *Event) Trigger(name string, args ...interface{}) {
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


