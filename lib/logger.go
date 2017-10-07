package lib

import (
    "fmt"
    "os"
    "time"
    "strings"
    "runtime"
    "path"
    "bytes"
)


type Logger struct {
    showCodeLine bool
    rotate string
    mute bool
    fileSuffix string
    dir string
    name string
    msgPipe chan []byte
    pipeLen int
    file *os.File
    inLoop bool
}

func NewLogger(dir, name, rotate string, showCodeLine bool) *Logger {
    this := &Logger{}
    this.dir = strings.TrimSuffix(dir, "/")
    this.name = strings.TrimPrefix(name, "/")
    this.pipeLen = 50
    this.msgPipe = make(chan []byte, this.pipeLen)
    this.rotate = rotate
    this.showCodeLine = showCodeLine
    this.fileSuffix = ".log"
    if err := this.StartLoop(); err != nil {
        panic(err)
    }
    return this
}

func (this *Logger) SetFileSuffix(s string) {
    this.fileSuffix = s
}

func (this *Logger) Name() string {
    return this.name
}

func (this *Logger) Mute() {
    this.mute = true
}

func (this *Logger) UnMute() {
    this.mute = false
}

func (this *Logger) Write(format string, args ...interface{}) {
    if this == nil {
        fmt.Printf("FATAL:[log module not init] "+format+"\n", args...)
        return
    }
    if this.mute {
        return
    }
    var head string
    now := time.Now()
    if this.showCodeLine {
        _, file, line, ok := runtime.Caller(1)
        if !ok {
            file = "???"
            line = 0
        } else {
            file = path.Base(file)
        }
        head = fmt.Sprintf("%s [%s:%d] ", now.Format("2006-01-02 15:04:05"), file, line)
    } else {
        head = fmt.Sprintf("%s ", now.Format("2006-01-02 15:04:05"))
    }
    if len(args) > 0 {
        this.msgPipe <- []byte(head + fmt.Sprintf(format, args...) + "\n")
    } else {
        this.msgPipe <- []byte(head + format + "\n")
    }
}

func (this *Logger) StartLoop() error {
    if err := this.openFile(); err != nil {
        return err
    }
    this.inLoop = true
    go this.flushLoop()
    return nil
}

func (this *Logger) StopLoop() {
    this.inLoop = false
}

func (this *Logger) flushLoop() {
    lineNum := 0
    cutTime := time.Now()
    for this.inLoop {
        var rows [][]byte
        select {
        case msg := <- this.msgPipe:
            msgLen := len(this.msgPipe)
            rows = make([][]byte, 0, msgLen + 1)
            rows = append(rows, msg)
            for i := 0; i < msgLen; i ++ {
                rows = append(rows, <-this.msgPipe)
            }
        case <- time.After(time.Minute):
        }
        if len(rows) > 0 {
            lineNum += len(rows)
            buffer := bytes.Join(rows, nil)
            if _, err := this.file.Write(buffer); err != nil {
                this.openFile()
                this.file.Write(buffer)
            }
        }
        now := time.Now()
        if lineNum > 0 && this.needRotate(now, cutTime) {
            this.file.Close()
            filename := fmt.Sprintf("%s-%s", this.Filename(), now.Add(-5 * time.Minute).Format("2006010215"))
            os.Rename(this.file.Name(), filename)
            this.openFile()
            lineNum = 0
            cutTime = now
        }
    }
}

func (this *Logger) Filename() string {
    return this.dir + "/" + this.name + this.fileSuffix
}

func (this *Logger) needRotate(now, cutTime time.Time) bool {
    if this.rotate == "daily" {
        return now.Hour() == 0 && now.Sub(cutTime).Hours() > 1
    }
    if this.rotate == "hourly" {
        return now.Minute() < 2 && now.Sub(cutTime).Minutes() > 2
    }
    return false
}

func (this *Logger) openFile() (err error) {
    this.file, err = os.OpenFile(this.Filename(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
    return
}


