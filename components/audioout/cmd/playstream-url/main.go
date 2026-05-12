// Package main is a minimal end-user example of PlayStream: stream raw PCM16
// audio from an HTTP URL to an audio_out resource. The producer (HTTP body
// reader) and the consumer (PlayStream) run concurrently — playback starts as
// soon as the first chunk arrives.
//
// The HTTP source can be anything that returns raw PCM16 bytes: a TTS provider's
// streaming endpoint, a static file served over HTTP, a podcast stream, etc.
//
//	go run ./components/audioout/cmd/playstream-url \
//	    -addr xarm-main.abc123.viam.cloud \
//	    -api-key-id <id> -api-key <key> \
//	    -name audio_out-1 \
//	    -url https://example.com/audio.pcm
package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.viam.com/utils/rpc"

	"go.viam.com/rdk/components/audioout"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/robot/client"
	"go.viam.com/rdk/utils"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "robot address")
	name := flag.String("name", "audio_out1", "audio_out resource name")
	apiKeyID := flag.String("api-key-id", "", "Viam API key ID (cloud machines)")
	apiKey := flag.String("api-key", "", "Viam API key (cloud machines)")
	url := flag.String("url", "", "HTTP URL serving raw PCM16 audio")
	sampleRate := flag.Int("sample-rate", 24000, "sample rate in Hz")
	numChannels := flag.Int("channels", 1, "number of channels")
	flag.Parse()

	if *url == "" {
		flag.Usage()
		os.Exit(2)
	}

	logger := logging.NewLogger("playstream-url")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var opts []client.RobotClientOption
	if *apiKeyID != "" && *apiKey != "" {
		opts = append(opts, client.WithDialOptions(rpc.WithEntityCredentials(
			*apiKeyID,
			rpc.Credentials{Type: rpc.CredentialsTypeAPIKey, Payload: *apiKey},
		)))
	}

	robot, err := client.New(ctx, *addr, logger, opts...)
	if err != nil {
		logger.Fatal(err)
	}
	defer robot.Close(ctx) //nolint:errcheck

	ao, err := audioout.FromProvider(robot, *name)
	if err != nil {
		logger.Fatalf("could not get audio_out %q: %v", *name, err)
	}

	info := &utils.AudioInfo{
		Codec:        utils.CodecPCM16,
		SampleRateHz: int32(*sampleRate),
		NumChannels:  int32(*numChannels),
	}

	// Producer: pull bytes from the HTTP body and push onto the channel.
	// Closing the channel signals end-of-stream to PlayStream.
	chunks := make(chan []byte)
	go func() {
		defer close(chunks)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, *url, nil)
		if err != nil {
			logger.Errorf("build request: %v", err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			logger.Errorf("http get: %v", err)
			return
		}
		defer resp.Body.Close()

		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				select {
				case chunks <- append([]byte(nil), buf[:n]...):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	logger.Infof("streaming %s to %q (%dHz, %dch)", *url, *name, *sampleRate, *numChannels)
	start := time.Now()
	if err := ao.PlayStream(ctx, info, chunks, nil); err != nil {
		logger.Fatalf("PlayStream failed: %v", err)
	}
	logger.Infof("PlayStream returned after %s", time.Since(start))
}
