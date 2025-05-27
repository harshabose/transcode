package transcode

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/asticode/go-astiav"
)

func TestTranscoderWithAVFoundation(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create demuxer with AVFoundation input format
	// Using "0" as input for facetime camera
	demuxer, err := CreateGeneralDemuxer(ctx, "0", WithAvFoundationInputFormatOption)
	if err != nil {
		t.Fatalf("Failed to create demuxer: %v", err)
	}

	// Create decoder
	decoder, err := CreateGeneralDecoder(ctx, demuxer)
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}

	// Create filter with video configuration
	filter, err := CreateGeneralFilter(ctx, decoder, VideoFilters,
		WithVideoScaleFilterContent(640, 480),
		WithVideoPixelFormatFilterContent(astiav.PixelFormatYuv420P),
		WithVideoFPSFilterContent(30))
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	// Create encoder with H.264 codec
	encoder, err := CreateGeneralEncoder(ctx, astiav.CodecIDH264, filter, WithX264LowLatencyOptions)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	// Create transcoder
	transcoder := NewTranscoder(demuxer, decoder, filter, encoder)

	// Start the transcoder
	transcoder.Start()
	defer time.Sleep(2 * time.Second)
	defer transcoder.Stop()

	// Wait for and process some packets to verify it's working
	fmt.Println("Transcoder started, waiting for packets...")

	packetCount := 0
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			// Test passed if we received some packets
			if packetCount > 0 {
				fmt.Printf("Test passed: received %d packets\n", packetCount)
				return
			}
			t.Fatalf("Timeout reached without receiving any packets")
		case packet := <-transcoder.WaitForPacket():
			packetCount++
			fmt.Printf("Received packet %d, size: %d bytes\n", packetCount, packet.Size())
			transcoder.PutBack(packet)

			// Exit after receiving a few packets
			if packetCount >= 10 {
				fmt.Printf("Test passed: received %d packets\n", packetCount)
				return
			}
		}
	}
}

func TestTranscoderWithEncoderBuilder(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create demuxer with AVFoundation input format
	// Using "0" as input for facetime camera
	demuxer, err := CreateGeneralDemuxer(ctx, "0", WithAvFoundationInputFormatOption)
	if err != nil {
		t.Fatalf("Failed to create demuxer: %v", err)
	}

	// Create decoder
	decoder, err := CreateGeneralDecoder(ctx, demuxer)
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}

	// Create filter with video configuration
	filter, err := CreateGeneralFilter(ctx, decoder, VideoFilters,
		WithVideoScaleFilterContent(640, 480),
		WithVideoPixelFormatFilterContent(astiav.PixelFormatYuv420P),
		WithVideoFPSFilterContent(30))
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	// Create encoder with H.264 codec using EncoderBuilder
	encoderBuilder := NewEncoderBuilder(astiav.CodecIDH264, &LowLatencyX264Settings, 10, filter)
	encoder, err := encoderBuilder.Build(ctx)
	if err != nil {
		t.Fatalf("Failed to create encoder: %v", err)
	}

	// Create transcoder
	transcoder := NewTranscoder(demuxer, decoder, filter, encoder)

	// Start the transcoder
	transcoder.Start()
	defer time.Sleep(2 * time.Second)
	defer transcoder.Stop()

	// Wait for and process some packets to verify it's working
	fmt.Println("Transcoder with EncoderBuilder started, waiting for packets...")

	packetCount := 0
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			// Test passed if we received some packets
			if packetCount > 0 {
				fmt.Printf("Test passed: received %d packets\n", packetCount)
				return
			}
			t.Fatalf("Timeout reached without receiving any packets")
		case packet := <-transcoder.WaitForPacket():
			packetCount++
			fmt.Printf("Received packet %d, size: %d bytes\n", packetCount, packet.Size())
			transcoder.PutBack(packet)

			// Exit after receiving a few packets
			if packetCount >= 10 {
				fmt.Printf("Test passed: received %d packets\n", packetCount)
				return
			}
		}
	}
}

func TestTranscoderWithUpdateEncoder(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create demuxer with AVFoundation input format
	// Using "0" as input for facetime camera
	demuxer, err := CreateGeneralDemuxer(ctx, "0", WithAvFoundationInputFormatOption)
	if err != nil {
		t.Fatalf("Failed to create demuxer: %v", err)
	}

	// Create decoder
	decoder, err := CreateGeneralDecoder(ctx, demuxer)
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}

	// Create filter with video configuration
	filter, err := CreateGeneralFilter(ctx, decoder, VideoFilters,
		WithVideoScaleFilterContent(640, 480),
		WithVideoPixelFormatFilterContent(astiav.PixelFormatYuv420P),
		WithVideoFPSFilterContent(30))
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	// Create encoder builder with WebRTCOptimisedX264Settings
	encoderBuilder := NewEncoderBuilder(astiav.CodecIDH264, &WebRTCOptimisedX264Settings, 10, filter)

	// Define min and max bitrates for testing (in bits per second)
	minBitrate := int64(500_000)   // 500 kbps
	maxBitrate := int64(1_500_000) // 1.5 Mbps

	// Create UpdateEncoder with configuration
	updateConfig := UpdateConfig{
		MinBitrate:              minBitrate,
		MaxBitrate:              maxBitrate,
		CutVideoBelowMinBitrate: false, // Don't pause when below min bitrate
	}
	fmt.Println("Trying to create updateEncoder...")

	updateEncoder, err := NewUpdateEncoder(ctx, updateConfig, encoderBuilder)
	if err != nil {
		t.Fatalf("Failed to create update encoder: %v", err)
	}

	fmt.Println("Created updateEncoder successfully...")

	// Create transcoder
	transcoder := NewTranscoder(demuxer, decoder, filter, updateEncoder)

	// Start the transcoder
	transcoder.Start()
	defer time.Sleep(2 * time.Second)
	defer transcoder.Stop()

	// Wait for and process some packets to verify it's working
	fmt.Println("Transcoder with UpdateEncoder started, waiting for packets...")

	// Define bitrates to test (in bits per second)
	bitrateTests := []struct {
		name    string
		bitrate int64
	}{
		{"Initial", 800_000},          // Initial bitrate (within range)
		{"Within range 1", 1_000_000}, // Within range
		{"Within range 2", 1_200_000}, // Within range
		{"Above max", 2_000_000},      // Above max (should be capped)
		{"Below min", 300_000},        // Below min (should be capped, not paused)
	}

	// Function to wait for packets after bitrate change
	waitForPackets := func(name string, count int) error {
		receivedCount := 0
		timeout := time.After(3 * time.Second)

		for receivedCount < count {
			select {
			case <-timeout:
				return fmt.Errorf("timeout waiting for packets after %s bitrate change", name)
			case packet := <-transcoder.WaitForPacket():
				receivedCount++
				fmt.Printf("[%s] Received packet %d, size: %d bytes\n", name, receivedCount, packet.Size())
				fmt.Println("Trying to putback packet")
				transcoder.PutBack(packet)
			}
		}
		return nil
	}

	// Test each bitrate
	for _, test := range bitrateTests {
		fmt.Printf("Updating bitrate to %d bps (%s)...\n", test.bitrate, test.name)

		err := transcoder.UpdateBitrate(test.bitrate)
		if err != nil {
			t.Fatalf("Failed to update bitrate to %d bps (%s): %v", test.bitrate, test.name, err)
		}
		fmt.Printf("Update to %d successfull\n", test.bitrate)

		// Wait for packets after bitrate change
		if err := waitForPackets(test.name, 5); err != nil {
			t.Fatal(err)
		}

		fmt.Printf("Successfully received packets after %s bitrate change\n", test.name)
	}

	fmt.Println("Test passed: successfully updated bitrate multiple times")
}

func TestTranscoderWithUpdateEncoderAndPausing(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Create demuxer with AVFoundation input format
	// Using "0" as input for facetime camera
	demuxer, err := CreateGeneralDemuxer(ctx, "0", WithAvFoundationInputFormatOption)
	if err != nil {
		t.Fatalf("Failed to create demuxer: %v", err)
	}

	// Create decoder
	decoder, err := CreateGeneralDecoder(ctx, demuxer)
	if err != nil {
		t.Fatalf("Failed to create decoder: %v", err)
	}

	// Create filter with video configuration
	filter, err := CreateGeneralFilter(ctx, decoder, VideoFilters,
		WithVideoScaleFilterContent(640, 480),
		WithVideoPixelFormatFilterContent(astiav.PixelFormatYuv420P),
		WithVideoFPSFilterContent(30))
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	// Create encoder builder with WebRTCOptimisedX264Settings
	encoderBuilder := NewEncoderBuilder(astiav.CodecIDH264, &WebRTCOptimisedX264Settings, 10, filter)

	// Define min and max bitrates for testing (in bits per second)
	minBitrate := int64(500_000)   // 500 kbps
	maxBitrate := int64(1_500_000) // 1.5 Mbps

	// Create UpdateEncoder with configuration that enables pausing
	updateConfig := UpdateConfig{
		MinBitrate:              minBitrate,
		MaxBitrate:              maxBitrate,
		CutVideoBelowMinBitrate: true, // Pause when below min bitrate
	}

	updateEncoder, err := NewUpdateEncoder(ctx, updateConfig, encoderBuilder)
	if err != nil {
		t.Fatalf("Failed to create update encoder: %v", err)
	}

	// Create transcoder
	transcoder := NewTranscoder(demuxer, decoder, filter, updateEncoder)

	// Start the transcoder
	transcoder.Start()
	defer time.Sleep(2 * time.Second)
	defer transcoder.Stop()

	// Wait for and process some packets to verify it's working
	fmt.Println("Transcoder with UpdateEncoder and pausing started, waiting for packets...")

	// Function to wait for packets with timeout
	waitForPackets := func(name string, count int) error {
		receivedCount := 0
		timeout := time.After(3 * time.Second)

		for receivedCount < count {
			select {
			case <-timeout:
				return fmt.Errorf("timeout waiting for packets after %s bitrate change", name)
			case packet := <-transcoder.WaitForPacket():
				receivedCount++
				fmt.Printf("[%s] Received packet %d, size: %d bytes\n", name, receivedCount, packet.Size())
				transcoder.PutBack(packet)
			}
		}

		fmt.Printf("Successfully received %d packets after %s bitrate change\n", count, name)
		return nil
	}

	// Function to test that WaitForPacket blocks when paused
	testPauseBlocking := func(name string) error {
		fmt.Printf("[%s] Testing that WaitForPacket blocks when paused...\n", name)

		// Start a goroutine to call WaitForPacket
		packetReceived := make(chan bool, 1)
		go func() {
			select {
			case <-transcoder.WaitForPacket():
				packetReceived <- true
			}
		}()

		// Wait briefly to see if packet is received (it shouldn't be)
		select {
		case <-packetReceived:
			return fmt.Errorf("received packet when encoding should be paused")
		case <-time.After(1 * time.Second):
			fmt.Printf("[%s] Confirmed: WaitForPacket is blocking (encoder paused)\n", name)
			return nil
		}
	}

	// First test with normal bitrate (should receive packets)
	fmt.Println("Testing with normal bitrate (800 kbps)...")
	if err := waitForPackets("Normal bitrate", 5); err != nil {
		t.Fatal(err)
	}

	// Update to below min bitrate with pausing enabled
	belowMinBitrate := int64(300_000) // 300 kbps
	fmt.Printf("Updating bitrate to %d bps (below min with pausing)...\n", belowMinBitrate)

	// Update bitrate in a goroutine since it might block if the encoder is processing
	updateComplete := make(chan error, 1)
	go func() {
		updateComplete <- transcoder.UpdateBitrate(belowMinBitrate)
	}()

	// Wait for update to complete
	select {
	case err := <-updateComplete:
		if err != nil {
			t.Fatalf("Failed to update bitrate to %d bps: %v", belowMinBitrate, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for bitrate update to complete")
	}

	// Test that WaitForPacket blocks when paused
	if err := testPauseBlocking("Below min with pausing"); err != nil {
		t.Fatal(err)
	}

	// Update back to normal bitrate (this should unpause)
	normalBitrate := int64(800_000) // 800 kbps
	fmt.Printf("Updating bitrate back to %d bps (normal)...\n", normalBitrate)

	// Update bitrate in a goroutine
	go func() {
		updateComplete <- transcoder.UpdateBitrate(normalBitrate)
	}()

	// Wait for update to complete
	select {
	case err := <-updateComplete:
		if err != nil {
			t.Fatalf("Failed to update bitrate to %d bps: %v", normalBitrate, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for bitrate update to complete")
	}

	// Wait for packets after bitrate change (should receive packets again)
	if err := waitForPackets("Back to normal", 5); err != nil {
		t.Fatal(err)
	}

	fmt.Println("Test passed: successfully tested pausing with low bitrate")
}
