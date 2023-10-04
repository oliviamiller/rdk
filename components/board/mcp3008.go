package board

import (
	"context"

	"go.uber.org/multierr"
	"go.viam.com/utils"
)

// MCP3008AnalogReader implements a board.AnalogReader using an MCP3008 ADC via SPI.
type MCP3008AnalogReader struct {
	Channel int
	Bus     SPI
	Chip    string
}

// AnalogConfig describes the configuration of an analog reader on a board.
// TODO RSDK-5177 : move this config to generic linux and pi.
type MCP3008AnalogConfig struct {
	Name              string `json:"name"`
	Pin               string `json:"pin"`         // analog input pin on the ADC itself
	SPIBus            string `json:"spi_bus"`     // name of the SPI bus (which is configured elsewhere in the config file)
	ChipSelect        string `json:"chip_select"` // the CS line for the ADC chip, typically a pin number on the board
	AverageOverMillis int    `json:"average_over_ms,omitempty"`
	SamplesPerSecond  int    `json:"samples_per_sec,omitempty"`
}

// Validate ensures all parts of the config are valid.
func (config *MCP3008AnalogConfig) Validate(path string) error {
	if config.Name == "" {
		return utils.NewConfigValidationFieldRequiredError(path, "name")
	}
	return nil
}

func (mar *MCP3008AnalogReader) Read(ctx context.Context, extra map[string]interface{}) (value int, err error) {
	var tx [3]byte
	tx[0] = 1                            // start bit
	tx[1] = byte((8 + mar.Channel) << 4) // single-ended
	tx[2] = 0                            // extra clocks to receive full 10 bits of data

	bus, err := mar.Bus.OpenHandle()
	if err != nil {
		return 0, err
	}
	defer func() {
		err = multierr.Combine(err, bus.Close())
	}()

	rx, err := bus.Xfer(ctx, 1000000, mar.Chip, 0, tx[:])
	if err != nil {
		return 0, err
	}
	// Reassemble the 10-bit value. Do not include bits before the final 10, because they contain
	// garbage and might be non-zero.
	val := 0x03FF & ((int(rx[1]) << 8) | int(rx[2]))

	return val, nil
}

// Close does nothing.
func (mar *MCP3008AnalogReader) Close(ctx context.Context) error {
	return nil
}
