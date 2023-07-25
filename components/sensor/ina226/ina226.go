// Package ina219 implements an ina219 voltage/current/power monitor sensor -
// typically used for battery state monitoring.
// Datasheet can be found at: https://www.ti.com/lit/ds/symlink/ina219.pdf
// Example repo: https://github.com/periph/devices/blob/main/ina219/ina219.go
package ina226

import (
	"context"
	"fmt"
	"log"

	"github.com/d2r2/go-i2c"
	gologger "github.com/d2r2/go-logger"
	"github.com/edaniels/golog"
	"go.viam.com/utils"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/resource"
)

var model = resource.DefaultModelFamily.WithModel("ina226")

const (
	milliAmp                     = 1000 * 1000 // milliAmp = 1000 microAmpere * 1000 nanoAmpere
	milliOhm                     = 1000 * 1000 // milliOhm = 1000 microOhm * 1000 nanoOhm
	defaultI2Caddr               = 0x40
	senseResistor        float64 = 100 * milliOhm                                                //.1 ohm
	maxCurrent           float64 = 20000 * milliAmp                                              // 20 amp
	calibratescale               = ((int64(1000*milliAmp) * int64(1000*milliOhm)) / 100000) << 9 // .00512 is internal fixed value in ina226
	configRegister               = 0x00
	shuntVoltageRegister         = 0x01
	busVoltageRegister           = 0x02
	powerRegister                = 0x03
	currentRegister              = 0x04
	calibrationRegister          = 0x05
)

// Config is used for converting config attributes.
type Config struct {
	I2CBus  int `json:"i2c_bus"`
	I2cAddr int `json:"i2c_addr,omitempty"`
}

// Validate ensures all parts of the config are valid.
func (conf *Config) Validate(path string) ([]string, error) {
	var deps []string
	if conf.I2CBus == 0 {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "i2c_bus")
	}
	return deps, nil
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		model,
		resource.Registration[sensor.Sensor, *Config]{
			Constructor: func(
				ctx context.Context,
				deps resource.Dependencies,
				conf resource.Config,
				logger golog.Logger,
			) (sensor.Sensor, error) {
				newConf, err := resource.NativeConfig[*Config](conf)
				if err != nil {
					return nil, err
				}
				return newSensor(deps, conf.ResourceName(), newConf, logger)
			},
		})
}

func newSensor(
	deps resource.Dependencies,
	name resource.Name,
	conf *Config,
	logger golog.Logger,
) (sensor.Sensor, error) {

	addr := conf.I2cAddr
	if addr == 0 {
		addr = defaultI2Caddr
		logger.Infof("using i2c address : %d", defaultI2Caddr)
	}

	s := &ina226{
		Named:  name.AsNamed(),
		logger: logger,
		addr:   byte(addr),
		bus:    conf.I2CBus,
	}

	err := s.calibrate()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// ina219 is a i2c sensor device that reports voltage, current and power.
type ina226 struct {
	resource.Named
	resource.AlwaysRebuild
	resource.TriviallyCloseable
	logger     golog.Logger
	addr       byte
	bus        int
	currentLSB float64
	powerLSB   float64
	cal        uint16
}

type powerMonitor struct {
	Shunt   int64
	Voltage float64
	Current float64
	Power   float64
}

func (d *ina226) calibrate() error {
	if senseResistor <= 0 {
		return fmt.Errorf("ina219 calibrate: senseResistor value invalid %f", senseResistor)
	}
	if maxCurrent <= 0 {
		return fmt.Errorf("ina219 calibrate: maxCurrent value invalid %f", maxCurrent)
	}

	d.currentLSB = maxCurrent / (1 << 15)
	d.powerLSB = 25 * d.currentLSB
	// Calibration Register = 0.04096 / (current LSB * Shunt Resistance)
	// Where lsb is in Amps and resistance is in ohms.
	// Calibration register is 16 bits.
	cal := float64(calibratescale) / (float64(d.currentLSB) * float64(senseResistor))
	log.Println("calibration")
	log.Println(cal)
	if cal >= (1 << 16) {
		return fmt.Errorf("ina219 calibrate: calibration register value invalid %f", cal)
	}
	d.cal = uint16(cal)

	return nil
}

// Readings returns a list containing three items (voltage, current, and power).
func (d *ina226) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {

	var err error
	handle, err := i2c.NewI2C(uint8(d.addr), d.bus)
	if err != nil {
		d.logger.Errorf("can't open ina226 i2c: %s", err)
		return nil, err
	}

	defer handle.Close()
	gologger.ChangePackageLogLevel("i2c", gologger.InfoLevel)

	// use the calibration result to set the scaling factor
	// of the current and power registers for the maximum resolution
	err = handle.WriteRegU16BE(calibrationRegister, d.cal)
	if err != nil {
		return nil, err
	}

	// setting config is 111 sets to normal operating mode
	err = handle.WriteRegU16BE(configRegister, uint16(0x6F))
	if err != nil {
		return nil, err
	}

	var pm powerMonitor

	// get shunt voltage - currently we are not returning - is it useful?
	shunt, err := handle.ReadRegU16BE(shuntVoltageRegister)
	if err != nil {
		return nil, err
	}

	// Least significant bit is 10µV.
	//pm.Shunt = int64(binary.BigEndian.Uint16(shunt)) * 10 * 1000
	pm.Shunt = int64(shunt) * 10 * 1000
	d.logger.Debugf("ina219 shunt : %d", pm.Shunt)

	bus, err := handle.ReadRegU16BE(busVoltageRegister)
	if err != nil {
		return nil, err
	}

	// bus voltage is 1.25 mV/bit
	pm.Voltage = float64(bus) * 1.25e-3

	current, err := handle.ReadRegU16BE(currentRegister)
	if err != nil {
		return nil, err
	}

	pm.Current = float64(current) * d.currentLSB / 1000000000

	power, err := handle.ReadRegU16BE(powerRegister)
	if err != nil {
		return nil, err
	}

	pm.Power = float64((power)) * d.powerLSB / 1000000000

	return map[string]interface{}{
		"volts": pm.Voltage,
		"amps":  pm.Current,
		"watts": pm.Power,
	}, nil
}
