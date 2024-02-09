package gpsnmea

import (
	"bufio"
	"context"
	"io"
	"sync"

	"github.com/jacobsa/go-serial/serial"
	"go.viam.com/utils"

	"go.viam.com/rdk/logging"
)

// DataReader represents a way to get data from a GPS NMEA device. We can read data from it using
// the channel in Messages, and we can close the device when we're done.
type DataReader interface {
	Messages() chan string
	Close() error
}

// SerialDataReader implements the DataReader interface by interacting with the device over a
// serial port.
type SerialDataReader struct {
	dev                     io.ReadWriteCloser
	data                    chan string
	cancelCtx               context.Context
	cancelFunc              func()
	activeBackgroundWorkers sync.WaitGroup
	logger                  logging.Logger
}

// NewSerialDataReader constructs a new DataReader that gets its NMEA messages over a serial port.
func NewSerialDataReader(options serial.OpenOptions, logger logging.Logger) (DataReader, error) {
	dev, err := serial.Open(options)
	if err != nil {
		return nil, err
	}

	data := make(chan string)
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	reader := SerialDataReader{
		dev:        dev,
		data:       data,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		logger:     logger,
	}
	reader.start()

	return &reader, nil
}

func (dr *SerialDataReader) start() {
	dr.activeBackgroundWorkers.Add(1)
	utils.PanicCapturingGo(func() {
		defer dr.activeBackgroundWorkers.Done()
		defer close(dr.data)

		r := bufio.NewReader(dr.dev)
		for {
			select {
			case <-dr.cancelCtx.Done():
				return
			default:
			}

			line, err := r.ReadString('\n')
			if err != nil {
				dr.logger.CErrorf(dr.cancelCtx, "can't read gps serial %s", err)
				continue // The line has bogus data; don't put it in the channel.
			}
			dr.data <- line
		}
	})
}

// Messages returns the channel of complete NMEA sentences we have read off of the device. It's part
// of the DataReader interface.
func (dr *SerialDataReader) Messages() chan string {
	return dr.data
}

// Close is part of the DataReader interface. It shuts everything down, including our connection to
// the serial port.
func (dr *SerialDataReader) Close() error {
	dr.cancelFunc()
	// If the background coroutine is trying to put a new line of data into the channel, it won't
	// notice that we've canceled it until something tries taking the line out of the channel. So,
	// let's try to read that out so the coroutine isn't stuck. If the background coroutine shut
	// itself down already, the channel will be closed and reading something out of it will just
	// return the empty string.
	<-dr.data
	dr.activeBackgroundWorkers.Wait()
	return dr.dev.Close()
}
