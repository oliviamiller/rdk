package main

import (
	"context"
	"fmt"
	"log"
	"time"

	streampb "go.viam.com/api/stream/v1"
	"go.viam.com/rdk/robot/client"
)

func main() {
	fmt.Println("Starting audio stream test client...")

	// Connect to your robot (adjust address as needed)
	robot, err := client.New(
		context.Background(),
		"localhost:8080", // Change this to your robot's address
		&client.Options{
			Insecure: true, // Remove this if using TLS
		},
	)
	if err != nil {
		log.Fatalf("Failed to connect to robot: %v", err)
	}
	defer robot.Close(context.Background())

	fmt.Println("Connected to robot successfully!")

	// Get streaming service
	streamClient := streampb.NewStreamServiceClient(robot)

	// List available streams
	fmt.Println("Listing available streams...")
	streams, err := streamClient.ListStreams(context.Background(), &streampb.ListStreamsRequest{})
	if err != nil {
		log.Fatalf("Failed to list streams: %v", err)
	}
	fmt.Printf("Available streams: %v\n", streams.Names)

	// Look for your audio stream
	audioStreamName := "audio-1" // Adjust this to match your audio resource name
	found := false
	for _, name := range streams.Names {
		if name == audioStreamName {
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("Audio stream '%s' not found in available streams. Available: %v\n", audioStreamName, streams.Names)
		fmt.Println("Make sure your audio component is configured and running.")
		return
	}

	fmt.Printf("Found audio stream: %s\n", audioStreamName)

	// Add the audio stream
	fmt.Printf("Adding audio stream: %s\n", audioStreamName)
	_, err = streamClient.AddStream(context.Background(), &streampb.AddStreamRequest{
		Name: audioStreamName,
	})
	if err != nil {
		log.Fatalf("Failed to add audio stream: %v", err)
	}

	fmt.Println("✅ Audio stream added successfully!")
	fmt.Println("WebRTC audio track should now be active and streaming.")

	// Keep the stream running for a bit to test
	fmt.Println("Keeping stream active for 30 seconds...")
	time.Sleep(30 * time.Second)

	// Remove the stream
	fmt.Printf("Removing audio stream: %s\n", audioStreamName)
	_, err = streamClient.RemoveStream(context.Background(), &streampb.RemoveStreamRequest{
		Name: audioStreamName,
	})
	if err != nil {
		log.Printf("Failed to remove audio stream: %v", err)
	} else {
		fmt.Println("✅ Audio stream removed successfully!")
	}

	fmt.Println("Test completed!")
}
