package logic

import (
    "fmt"
    "sync"
    "time"
)

type Counter struct {
    total int
    filtered int
    fail int
    success int
    partSuccess int

    startTs time.Time
    finishTs time.Time

    mutex sync.Mutex
}

func (counter *Counter) setTotal(t int) {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    counter.total = t
}

func (counter *Counter) addFiltered() {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    counter.filtered++
}

func (counter *Counter) addFail() {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    counter.fail++
}

func (counter *Counter) addSuccess(full bool) {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    if full {
        counter.success++
    } else {
        counter.partSuccess++
    }
}

func (counter *Counter) start() {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    counter.startTs = time.Now()
}

func (counter *Counter) stop() {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    counter.finishTs = time.Now()
}

func (counter *Counter) printReport() {
    counter.mutex.Lock()
    defer counter.mutex.Unlock()

    fmt.Println("\tFiles:")
    fmt.Println("Total:...........................", counter.total)
    fmt.Println("Filtered:........................", counter.filtered)
    fmt.Println("Failed to read or recognize:.....", counter.fail)
    fmt.Println("Recognized without cover:........", counter.partSuccess)
    fmt.Println("Fully recognized:................", counter.success)
    fmt.Println("Time elapsed:....................", counter.finishTs.Sub(counter.startTs))
}
