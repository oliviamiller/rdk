// Package board contains a gRPC based board client.
package board

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/pkg/errors"
	commonpb "go.viam.com/api/common/v1"
	pb "go.viam.com/api/component/board/v1"
	"go.viam.com/utils"
	"go.viam.com/utils/protoutils"
	"go.viam.com/utils/rpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"

	"go.viam.com/rdk/logging"
	rprotoutils "go.viam.com/rdk/protoutils"
	"go.viam.com/rdk/resource"
)

// errUnimplemented is used for any unimplemented methods that should
// eventually be implemented server side or faked client side.
var errUnimplemented = errors.New("unimplemented")

// client implements BoardServiceClient.
type client struct {
	resource.Named
	resource.TriviallyReconfigurable
	resource.TriviallyCloseable
	client pb.BoardServiceClient
	logger logging.Logger

	info           boardInfo
	cachedStatus   *commonpb.BoardStatus
	cachedStatusMu sync.Mutex

	streamCancel  context.CancelFunc
	streamRunning bool
	streamReady   bool
	streamMu      sync.Mutex
	mu            sync.RWMutex

	closeContext            context.Context
	activeBackgroundWorkers sync.WaitGroup
	cancelBackgroundWorkers context.CancelFunc
	extra                   *structpb.Struct

	callbacks map[string]chan Tick
}

type boardInfo struct {
	name                  string
	analogReaderNames     []string
	digitalInterruptNames []string
}

// NewClientFromConn constructs a new Client from connection passed in.
func NewClientFromConn(
	ctx context.Context,
	conn rpc.ClientConn,
	remoteName string,
	name resource.Name,
	logger logging.Logger,
) (Board, error) {
	info := boardInfo{name: name.ShortName()}
	bClient := pb.NewBoardServiceClient(conn)
	c := &client{
		Named:        name.PrependRemote(remoteName).AsNamed(),
		client:       bClient,
		logger:       logger,
		info:         info,
		closeContext: ctx,
	}
	if err := c.refresh(ctx); err != nil {
		c.logger.CWarn(ctx, err)
	}
	return c, nil
}

func (c *client) AnalogReaderByName(name string) (AnalogReader, bool) {
	return &analogReaderClient{
		client:           c,
		boardName:        c.info.name,
		analogReaderName: name,
	}, true
}

func (c *client) DigitalInterruptByName(name string) (DigitalInterrupt, bool) {
	return &digitalInterruptClient{
		client:               c,
		boardName:            c.info.name,
		digitalInterruptName: name,
	}, true
}

func (c *client) GPIOPinByName(name string) (GPIOPin, error) {
	return &gpioPinClient{
		client:    c,
		boardName: c.info.name,
		pinName:   name,
	}, nil
}

func (c *client) AnalogReaderNames() []string {
	if c.getCachedStatus() == nil {
		c.logger.Debugw("no cached status")
		return []string{}
	}
	return copyStringSlice(c.info.analogReaderNames)
}

func (c *client) DigitalInterruptNames() []string {
	if c.getCachedStatus() == nil {
		c.logger.Debugw("no cached status")
		return []string{}
	}
	return copyStringSlice(c.info.digitalInterruptNames)
}

// Status uses the cached status or a newly fetched board status to return the state
// of the board.
func (c *client) Status(ctx context.Context, extra map[string]interface{}) (*commonpb.BoardStatus, error) {
	if status := c.getCachedStatus(); status != nil {
		return status, nil
	}

	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Status(ctx, &pb.StatusRequest{Name: c.info.name, Extra: ext})
	if err != nil {
		return nil, err
	}
	return resp.Status, nil
}

func (c *client) refresh(ctx context.Context) error {
	status, err := c.status(ctx)
	if err != nil {
		return errors.Wrap(err, "status call failed")
	}
	c.storeStatus(status)

	c.info.analogReaderNames = []string{}
	for name := range status.Analogs {
		c.info.analogReaderNames = append(c.info.analogReaderNames, name)
	}
	c.info.digitalInterruptNames = []string{}
	for name := range status.DigitalInterrupts {
		c.info.digitalInterruptNames = append(c.info.digitalInterruptNames, name)
	}

	return nil
}

// storeStatus atomically stores the status response.
func (c *client) storeStatus(status *commonpb.BoardStatus) {
	c.cachedStatusMu.Lock()
	defer c.cachedStatusMu.Unlock()
	c.cachedStatus = status
}

// getCachedStatus atomically gets the cached status response.
func (c *client) getCachedStatus() *commonpb.BoardStatus {
	c.cachedStatusMu.Lock()
	defer c.cachedStatusMu.Unlock()
	return c.cachedStatus
}

// status gets the latest status from the server.
func (c *client) status(ctx context.Context) (*commonpb.BoardStatus, error) {
	resp, err := c.client.Status(ctx, &pb.StatusRequest{Name: c.info.name})
	if err != nil {
		return nil, err
	}
	return resp.Status, nil
}

func (c *client) SetPowerMode(ctx context.Context, mode pb.PowerMode, duration *time.Duration) error {
	var dur *durationpb.Duration
	if duration != nil {
		dur = durationpb.New(*duration)
	}
	_, err := c.client.SetPowerMode(ctx, &pb.SetPowerModeRequest{Name: c.info.name, PowerMode: mode, Duration: dur})
	return err

}

func (c *client) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return rprotoutils.DoFromResourceClient(ctx, c.client, c.info.name, cmd)
}

// WriteAnalog writes the analog value to the specified pin.
func (c *client) WriteAnalog(ctx context.Context, pin string, value int32, extra map[string]interface{}) error {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return err
	}
	_, err = c.client.WriteAnalog(ctx, &pb.WriteAnalogRequest{
		Name:  c.info.name,
		Pin:   pin,
		Value: value,
		Extra: ext,
	})

	return err
}

// analogReaderClient satisfies a gRPC based board.AnalogReader. Refer to the interface
// for descriptions of its methods.
type analogReaderClient struct {
	*client
	boardName        string
	analogReaderName string
}

func (arc *analogReaderClient) Read(ctx context.Context, extra map[string]interface{}) (int, error) {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return 0, err
	}
	resp, err := arc.client.client.ReadAnalogReader(ctx, &pb.ReadAnalogReaderRequest{
		BoardName:        arc.boardName,
		AnalogReaderName: arc.analogReaderName,
		Extra:            ext,
	})
	if err != nil {
		return 0, err
	}
	return int(resp.Value), nil
}

// digitalInterruptClient satisfies a gRPC based board.DigitalInterrupt. Refer to the
// interface for descriptions of its methods.
type digitalInterruptClient struct {
	*client
	boardName            string
	digitalInterruptName string
}

func (dic *digitalInterruptClient) Value(ctx context.Context, extra map[string]interface{}) (int64, error) {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return 0, err
	}
	resp, err := dic.client.client.GetDigitalInterruptValue(ctx, &pb.GetDigitalInterruptValueRequest{
		BoardName:            dic.boardName,
		DigitalInterruptName: dic.digitalInterruptName,
		Extra:                ext,
	})
	if err != nil {
		return 0, err
	}
	return resp.Value, nil
}

func (dic *digitalInterruptClient) Tick(ctx context.Context, high bool, nanoseconds uint64) error {
	tick := Tick{Name: dic.digitalInterruptName, High: high, TimestampNanosec: nanoseconds}
	dic.callbacks[dic.digitalInterruptName] <- tick
	return nil
}

func (dic *digitalInterruptClient) AddCallback(ctx context.Context, ch chan Tick, extra map[string]interface{}) error {
	return nil
}

// func (dic *digitalInterruptClient) AddCallback(ctx context.Context, interrupts []string, ch chan Tick, extra map[string]interface{}) error {
// 	fmt.Println("Digital interrupt client add a callback")
// 	dic.mu.Lock()
// 	if dic.callbacks == nil {
// 		dic.callbacks = make(map[string]chan Tick)
// 	}

// 	dic.callbacks[dic.digitalInterruptName] = ch
// 	dic.mu.Unlock()

// 	// We want only one ticks stream per board
// 	dic.streamMu.Lock()
// 	defer dic.streamMu.Unlock()
// 	ext, err := protoutils.StructToStructPb(extra)
// 	if err != nil {
// 		return err
// 	}
// 	dic.extra = ext

// 	fmt.Println("stream running?")
// 	fmt.Println(dic.streamRunning)
// 	// Only want one of these so cancel it if we're adding a new callback
// 	if dic.streamRunning {
// 		fmt.Println("stream already running canceling")
// 		for !dic.streamReady {
// 			if !utils.SelectContextOrWait(ctx, 50*time.Millisecond) {
// 				return ctx.Err()
// 			}
// 		}
// 		dic.streamReady = false
// 		fmt.Println("here - canceling stream")
// 		dic.streamCancel()
// 		// dic.cancelBackgroundWorkers()
// 		// dic.activeBackgroundWorkers.Wait()
// 		fmt.Println("here done")
// 	}
// 	// Start the stream
// 	dic.streamRunning = true
// 	dic.activeBackgroundWorkers.Add(1)
// 	closeContext, cancel := context.WithCancel(dic.closeContext)
// 	dic.cancelBackgroundWorkers = cancel
// 	// Create a background go routine that connects to the server's stream
// 	utils.PanicCapturingGo(func() {
// 		dic.connectTickStream(closeContext)
// 	})
// 	dic.mu.RLock()
// 	ready := dic.streamReady
// 	dic.mu.RUnlock()

// 	// wait until the stream is ready to return
// 	for !ready {
// 		fmt.Println("here - stream not ready yet")
// 		dic.mu.RLock()
// 		ready = dic.streamReady
// 		dic.mu.RUnlock()
// 		// wait for 50 ms before checking again
// 		if !utils.SelectContextOrWait(ctx, 50*time.Millisecond) {
// 			fmt.Println("context here")
// 			return ctx.Err()
// 		}
// 	}
// 	fmt.Println("returning nil - stream ready")
// 	return nil
// }

func (c *client) StartTickStream(ctx context.Context, interrupts []string, ch chan Tick, extra map[string]interface{}) error {

	if c.callbacks == nil {
		c.callbacks = make(map[string]chan Tick)
	}

	for _, interrupt := range interrupts {
		c.callbacks[interrupt] = ch
	}

	c.streamMu.Lock()
	defer c.streamMu.Unlock()
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return err
	}
	c.extra = ext

	// Start the stream
	c.streamRunning = true
	c.activeBackgroundWorkers.Add(1)
	_, cancel := context.WithCancel(ctx)
	c.cancelBackgroundWorkers = cancel
	// Create a background go routine that connects to the server's stream
	utils.PanicCapturingGo(func() {
		defer c.activeBackgroundWorkers.Done()
		c.connectTickStream(context.Background(), interrupts, ch)
	})
	c.mu.RLock()
	ready := c.streamReady
	c.mu.RUnlock()

	// wait until the stream is ready to return
	for !ready {
		fmt.Println("here - stream not ready yet")
		c.mu.RLock()
		ready = c.streamReady
		c.mu.RUnlock()
		// wait for 50 ms before checking again
		if !utils.SelectContextOrWait(ctx, 50*time.Millisecond) {
			fmt.Println("context here")
			return ctx.Err()
		}
	}
	fmt.Println("returning nil - stream ready")
	return nil

}

func (c *client) RemoveTickStream() error {
	fmt.Println("removing the tick stream")
	c.streamCancel()

	//c.cancelBackgroundWorkers()

	//c.activeBackgroundWorkers.Wait()
	fmt.Println("returning from remove tick stream")
	return nil
}

func (c *client) connectTickStream(ctx context.Context, interrupts []string, ch chan Tick) {
	defer func() {
		c.streamMu.Lock()
		defer c.streamMu.Unlock()
		c.mu.Lock()
		defer c.mu.Unlock()
		c.streamCancel = nil
		c.streamRunning = false
		c.streamReady = false
	}()

	streamCtx, cancel := context.WithCancel(ctx)
	c.streamCancel = cancel

	for {
		fmt.Println("connecting to the tick stream....")
		c.mu.Lock()
		c.streamReady = false
		c.mu.Unlock()
		select {
		// case <-c.closeContext.Done():
		// 	fmt.Println("close context is done in connect stream")
		// 	//c.streamCancel()
		// 	return
		// // case <-streamCtx.Done():
		// // 	fmt.Println("stream ctx done")
		// // 	return
		case <-ctx.Done():
			fmt.Println("ctx done")
			return
		default:
		}

		req := &pb.StreamTicksRequest{
			Name:       c.info.name,
			Interrupts: interrupts,
		}

		//call the server to start streaming ticks
		stream, err := c.client.StreamTicks(streamCtx, req)
		if err != nil {
			fmt.Println("error while calling stream ticks from client")
			c.logger.CError(ctx, err)
			if utils.SelectContextOrWait(ctx, 3*time.Second) {
				continue
			} else {
				fmt.Println("waited...returning")
				return
			}
		}

		// repeatly recieve from the stream
		for {
			select {
			case <-c.closeContext.Done():
				fmt.Println("close context is done in connect stream")
				//c.streamCancel()
				return
			// case <-streamCtx.Done():
			// 	fmt.Println("stream ctx done")
			// 	return
			// case <-ctx.Done():
			// 	fmt.Println("ctx done")
			// 	return
			default:
			}

			c.mu.Lock()
			c.streamReady = true
			c.mu.Unlock()
			streamResp, err := stream.Recv()
			if err != nil && streamResp == nil {
				fmt.Println("stream resp is nil and error")
				fmt.Println(err)
				//return
			}
			if err != nil {
				fmt.Println("err not nil and resp not nil")
				c.logger.CError(ctx, err)
			} else {
				// If there is a response with a tick, send it to the callback channel.
				tick := Tick{
					Name:             streamResp.Name,
					High:             streamResp.Tick.High,
					TimestampNanosec: streamResp.Tick.Time,
				}
				ch <- tick
			}
		}
	}

}

// func (c *client) connectTickStream(ctx context.Context) {
// 	defer func() {
// 		c.streamMu.Lock()
// 		defer c.streamMu.Unlock()
// 		c.mu.Lock()
// 		defer c.mu.Unlock()
// 		c.streamCancel = nil
// 		c.streamRunning = false
// 		c.streamReady = false
// 	}()

// 	for {
// 		fmt.Println("connecting to the tick stream....")
// 		c.mu.Lock()
// 		c.streamReady = false
// 		c.mu.Unlock()
// 		select {
// 		case <-ctx.Done():
// 			fmt.Println("context is done in connect stream")
// 			c.streamCancel()
// 			return
// 		default:
// 		}

// 		// Get all interrupts that have callbacks
// 		interrupts := make([]string, len(c.callbacks))
// 		i := 0
// 		for k := range c.callbacks {
// 			interrupts[i] = k
// 			i++
// 		}

// 		req := &pb.StreamTicksRequest{
// 			Name:       c.info.name,
// 			Interrupts: interrupts,
// 		}

// 		ctx, cancel := context.WithCancel(ctx)
// 		c.streamCancel = cancel

// 		// call the server to start streaming ticks
// 		stream, err := c.client.StreamTicks(ctx, req)
// 		if err != nil {
// 			fmt.Println("error while calling stream ticks from client")
// 			c.logger.CError(ctx, err)
// 			if utils.SelectContextOrWait(ctx, 3*time.Second) {
// 				continue
// 			} else {
// 				fmt.Println("waited...returning")
// 				return
// 			}
// 		}

// 		// repeatly recieve from the stream
// 		for {
// 			fmt.Println("in for loop reading stream")
// 			select {
// 			case <-ctx.Done():
// 				fmt.Println("ending.....context done in connectStream")
// 				return
// 			default:
// 			}

// 			c.mu.Lock()
// 			c.streamReady = true
// 			c.mu.Unlock()
// 			fmt.Println("recieving from stream")
// 			streamResp, err := stream.Recv()
// 			if err != nil && streamResp == nil {
// 				fmt.Println("stream resp is nil and error")
// 				fmt.Println(err)
// 				//c.activeBackgroundWorkers.Done()
// 				//return
// 			}
// 			if err != nil {
// 				fmt.Println("err not nil and resp not nil")
// 				c.logger.CError(ctx, err)
// 			} else {
// 				// If there is a response with a tick, send it to the callback channel.
// 				c.execCallback(ctx, streamResp)
// 			}
// 		}
// 	}
// }

func (c *client) execCallback(ctx context.Context, resp *pb.StreamTicksResponse) {
	// Convert proto tick to Go struct tick and send to the callback channel for that digital interrupt.
	tick := Tick{
		High:             resp.Tick.High,
		TimestampNanosec: resp.Tick.Time,
	}
	c.callbacks[resp.Name] <- tick
}

func (dic *digitalInterruptClient) RemoveCallback(c chan Tick) {
	fmt.Println("REMOVING CALLBACK RECLIENT")
	dic.mu.Lock()
	defer dic.mu.Unlock()
	delete(dic.callbacks, dic.digitalInterruptName)

	// If there are no more callbacks, stop streaming.
	if len(dic.callbacks) == 0 {
		fmt.Println("removing all callbacks....")
		if dic.cancelBackgroundWorkers != nil {
			dic.cancelBackgroundWorkers()
			dic.cancelBackgroundWorkers = nil
		}
		//dic.activeBackgroundWorkers.Wait()
	}
}

// Close cleanly closes the underlying connections.
func (dic *digitalInterruptClient) Close(ctx context.Context) error {
	if dic.cancelBackgroundWorkers != nil {
		dic.cancelBackgroundWorkers()
		dic.cancelBackgroundWorkers = nil
	}
	dic.activeBackgroundWorkers.Wait()
	return nil
}

// gpioPinClient satisfies a gRPC based board.GPIOPin. Refer to the interface
// for descriptions of its methods.
type gpioPinClient struct {
	*client
	boardName string
	pinName   string
}

func (gpc *gpioPinClient) Set(ctx context.Context, high bool, extra map[string]interface{}) error {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return err
	}
	_, err = gpc.client.client.SetGPIO(ctx, &pb.SetGPIORequest{
		Name:  gpc.boardName,
		Pin:   gpc.pinName,
		High:  high,
		Extra: ext,
	})
	return err
}

func (gpc *gpioPinClient) Get(ctx context.Context, extra map[string]interface{}) (bool, error) {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return false, err
	}
	resp, err := gpc.client.client.GetGPIO(ctx, &pb.GetGPIORequest{
		Name:  gpc.boardName,
		Pin:   gpc.pinName,
		Extra: ext,
	})
	if err != nil {
		return false, err
	}
	return resp.High, nil
}

func (gpc *gpioPinClient) PWM(ctx context.Context, extra map[string]interface{}) (float64, error) {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return math.NaN(), err
	}
	resp, err := gpc.client.client.PWM(ctx, &pb.PWMRequest{
		Name:  gpc.boardName,
		Pin:   gpc.pinName,
		Extra: ext,
	})
	if err != nil {
		return math.NaN(), err
	}
	return resp.DutyCyclePct, nil
}

func (gpc *gpioPinClient) SetPWM(ctx context.Context, dutyCyclePct float64, extra map[string]interface{}) error {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return err
	}
	_, err = gpc.client.client.SetPWM(ctx, &pb.SetPWMRequest{
		Name:         gpc.boardName,
		Pin:          gpc.pinName,
		DutyCyclePct: dutyCyclePct,
		Extra:        ext,
	})
	return err
}

func (gpc *gpioPinClient) PWMFreq(ctx context.Context, extra map[string]interface{}) (uint, error) {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return 0, err
	}
	resp, err := gpc.client.client.PWMFrequency(ctx, &pb.PWMFrequencyRequest{
		Name:  gpc.boardName,
		Pin:   gpc.pinName,
		Extra: ext,
	})
	if err != nil {
		return 0, err
	}
	return uint(resp.FrequencyHz), nil
}

func (gpc *gpioPinClient) SetPWMFreq(ctx context.Context, freqHz uint, extra map[string]interface{}) error {
	ext, err := protoutils.StructToStructPb(extra)
	if err != nil {
		return err
	}
	_, err = gpc.client.client.SetPWMFrequency(ctx, &pb.SetPWMFrequencyRequest{
		Name:        gpc.boardName,
		Pin:         gpc.pinName,
		FrequencyHz: uint64(freqHz),
		Extra:       ext,
	})
	return err
}

// copyStringSlice is a helper to simply copy a string slice
// so that no one mutates it.
func copyStringSlice(src []string) []string {
	out := make([]string, len(src))
	copy(out, src)
	return out
}
