package main

import (
	"io"
	"net/http"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

const FlushDelay = 100 * time.Millisecond

type PeriodicFlushWriter struct {
	writer    io.Writer
	flusher   http.Flusher
	timer     *time.Timer
	writeLock sync.Mutex
}

func NewPeriodicFlushWriter(writer io.Writer) *PeriodicFlushWriter {
	obj := PeriodicFlushWriter{writer: writer, writeLock: sync.Mutex{}}
	if f, ok := writer.(http.Flusher); ok {
		obj.flusher = f
		log.Debug("Flush supported")
	} else {
		log.Debug("Flush NOT supported")
	}
	obj.timer = time.NewTimer(FlushDelay)
	go func() {
		for {
			log.Debug("wait for timer being triggered")
			<-obj.timer.C
			log.Debug("Timer: fired, flushing..")
			obj.Flush()
		}
	}()
	return &obj
}

func (w *PeriodicFlushWriter) Flush() {
	if w.flusher != nil {
		w.writeLock.Lock()
		log.Debug("Flushing..")
		w.flusher.Flush()
		w.writeLock.Unlock()
	}
}

func (w *PeriodicFlushWriter) ResetFlushTimeout() {
	log.Debug("Reset timer")
	if !w.timer.Stop() {
		// <-w.timer.C
	}

	log.Debug("Reset timer, really now")
	w.timer.Reset(FlushDelay)
}

func (w *PeriodicFlushWriter) Close() {
	w.Flush()
	w.timer.Stop()
}

func (w *PeriodicFlushWriter) Write(p []byte) (int, error) {
	w.writeLock.Lock()
	defer w.writeLock.Unlock()
	n, err := w.writer.Write(p)
	w.ResetFlushTimeout()
	return n, err
}
