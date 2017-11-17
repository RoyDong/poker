package utils

import (
    "time"
    "os"
)

type FileLoader struct {
    Event
    filename      string
    lastModTime   time.Time
    checkInterval time.Duration
    inLoop        bool
}

func NewFileLoader(filename string, checkInterval time.Duration) *FileLoader {
    f := &FileLoader{
        filename:      filename,
        checkInterval: checkInterval,
    }
    return f
}

func (f *FileLoader) LoadFile() error {
    file, err := os.Open(f.filename)
    defer file.Close()
    if err != nil {
        return err
    }
    fileInfo, err := file.Stat()
    if err != nil {
        return err
    }
    if fileInfo.ModTime().UnixNano() > f.lastModTime.UnixNano() {
        buffer := make([]byte, fileInfo.Size())
        if _, err := file.Read(buffer); err != nil {
            return err
        }
        f.lastModTime = fileInfo.ModTime()
        f.Trigger("NewContent", buffer)
    }
    return nil
}

func (f *FileLoader) Filename() string {
    return f.filename
}

func (f *FileLoader) loadFileLoop() {
    for {
        time.Sleep(f.checkInterval)
        if !f.inLoop {
            return
        }
        f.LoadFile()
    }
}

func (f *FileLoader) StartLoadLoop() {
    if f.inLoop {
        return
    }
    f.inLoop = true
    go f.loadFileLoop()
}

func (f *FileLoader) StopLoadLoop() {
    f.inLoop = false
}
