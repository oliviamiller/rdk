// Package renogy implements the renogy charge controller sensor for DC batteries.
// Tested with renogy wanderer model
// Wanderer Manual: https://www.renogy.com/content/RNG-CTRL-WND30-LI/WND30-LI-Manual.pdf
// LCD Wanderer Manual: https://ca.renogy.com/content/manual/RNG-CTRL-WND10-Manual.pdf
package renogy

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/edaniels/golog"
	"github.com/goburrow/modbus"

	"go.viam.com/rdk/components/powersensor"
	"go.viam.com/rdk/resource"
)

var (
	model    = resource.DefaultModelFamily.WithModel("renogy")
	readings map[string]interface{}
)

const (
	// defaults assume the device is connected via UART serial.
	pathDefault     = "/dev/serial0"
	baudDefault     = 9600
	modbusIDDefault = 1

	solarVoltReg             = 263
	solarAmpReg              = 264
	solarWattReg             = 265
	loadVoltReg              = 260
	loadAmpReg               = 261
	loadWattReg              = 262
	battVoltReg              = 257
	battChargePctReg         = 256
	controllerDegCReg        = 259
	maxSolarTodayWattReg     = 271
	minSolarTodayWattReg     = 272
	maxBattTodayVoltReg      = 268
	minBattTodayVoltReg      = 267
	maxSolarTodayAmpReg      = 269
	minSolarTodayAmpReg      = 270
	chargeTodayWattHrsReg    = 273
	dischargeTodayWattHrsReg = 274
	chargeTodayAmpHrsReg     = 275
	dischargeTodayAmpHrsReg  = 276
	totalBattOverChargesReg  = 278
	totalBattFullChargesReg  = 279

	isAc = false
)

// Config is used for converting config attributes.
type Config struct {
	resource.TriviallyValidateConfig
	Path     string `json:"serial_path,omitempty"`
	Baud     int    `json:"serial_baud_rate,omitempty"`
	ModbusID byte   `json:"modbus_id,omitempty"`
}

func init() {
	resource.RegisterComponent(
		powersensor.API,
		model,
		resource.Registration[powersensor.PowerSensor, *Config]{
			Constructor: newRenogy,
		})
}

func newRenogy(_ context.Context, _ resource.Dependencies, conf resource.Config, logger golog.Logger) (powersensor.PowerSensor, error) {
	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return nil, err
	}

	if newConf.Path == "" {
		newConf.Path = pathDefault
	}
	if newConf.Baud == 0 {
		newConf.Baud = baudDefault
	}
	if newConf.ModbusID == 0 {
		newConf.ModbusID = modbusIDDefault
	}

	r := &Renogy{
		Named:    conf.ResourceName().AsNamed(),
		logger:   logger,
		path:     newConf.Path,
		baud:     newConf.Baud,
		modbusID: newConf.ModbusID,
	}

	r.handler = r.getHandler()

	err = r.handler.Connect()
	if err != nil {
		return nil, err
	}
	r.client = modbus.NewClient(r.handler)

	return r, nil
}

// Renogy is a serial charge controller.
type Renogy struct {
	resource.Named
	resource.AlwaysRebuild
	logger   golog.Logger
	mu       sync.Mutex
	path     string
	baud     int
	modbusID byte
	handler  *modbus.RTUClientHandler
	client   modbus.Client
}

// getHandler is a helper function to create the modbus handler.
func (r *Renogy) getHandler() *modbus.RTUClientHandler {
	handler := modbus.NewRTUClientHandler(r.path)
	handler.BaudRate = r.baud
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = r.modbusID
	handler.Timeout = 1 * time.Second
	return handler
}

// Voltage returns the voltage of the battery and a boolean IsAc.
func (r *Renogy) Voltage(ctx context.Context, extra map[string]interface{}) (float64, bool, error) {

	// Read the battery voltage.
	volts, err := r.readRegister(r.client, battVoltReg, 1)
	if err != nil {
		return 0, false, err
	}
	return float64(volts), isAc, nil
}

// Current returns the load's current and boolean isAC.
// If the controller does not have a load input, will return zero.
func (r *Renogy) Current(ctx context.Context, extra map[string]interface{}) (float64, bool, error) {

	// read the load current.
	loadCurrent, err := r.readRegister(r.client, loadAmpReg, 2)
	if err != nil {
		return 0, false, err
	}

	return float64(loadCurrent), isAc, nil
}

// Power returns the power of the load. If the controller does not have a load input, will return zero.
func (r *Renogy) Power(ctx context.Context, extra map[string]interface{}) (float64, error) {
	// reads the load wattage.
	loadPower, err := r.readRegister(r.client, loadWattReg, 1)
	fmt.Println(err)
	fmt.Println(loadPower)
	if err != nil {
		return 0, err
	}

	return float64(loadPower), err
}

// Readings returns a list of all readings from the sensor.
func (r *Renogy) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {

	readings = make(map[string]interface{})

	// add all readings.
	r.addReading(r.client, solarVoltReg, 1, "SolarVolt")
	r.addReading(r.client, solarAmpReg, 2, "SolarAmp")
	r.addReading(r.client, solarWattReg, 0, "SolarWatt")
	r.addReading(r.client, loadVoltReg, 1, "LoadVolt")
	r.addReading(r.client, loadAmpReg, 2, "LoadAmp")
	r.addReading(r.client, loadWattReg, 0, "LoadWatt")
	r.addReading(r.client, battVoltReg, 1, "BattVolt")
	r.addReading(r.client, battChargePctReg, 0, "BattChargePct")
	r.addReading(r.client, maxSolarTodayWattReg, 0, "MaxSolarTodayWatt")
	r.addReading(r.client, minSolarTodayWattReg, 0, "MinSolarTodayWatt")
	r.addReading(r.client, maxBattTodayVoltReg, 1, "MaxBattTodayVolt")
	r.addReading(r.client, minBattTodayVoltReg, 1, "MinBattTodayVolt")
	r.addReading(r.client, maxSolarTodayAmpReg, 2, "MaxSolarTodayAmp")
	r.addReading(r.client, minSolarTodayAmpReg, 1, "MinSolarTodayAmp")
	r.addReading(r.client, chargeTodayAmpHrsReg, 0, "ChargeTodayAmpHrs")
	r.addReading(r.client, dischargeTodayAmpHrsReg, 0, "DischargeTodayAmpHrs")
	r.addReading(r.client, chargeTodayWattHrsReg, 0, "ChargeTodayWattHrs")
	r.addReading(r.client, dischargeTodayWattHrsReg, 0, "DischargeTodayWattHrs")
	r.addReading(r.client, totalBattOverChargesReg, 0, "TotalBattOverCharges")
	r.addReading(r.client, totalBattFullChargesReg, 0, "TotalBattFullCharges")

	// Controller and battery temperates require math on controller deg register.
	tempReading, err := r.readRegister(r.client, controllerDegCReg, 0)
	if err != nil {
		return readings, err
	}

	battTempSign := (int16(tempReading) & 0b0000000010000000) >> 7
	battTemp := int16(tempReading) & 0b0000000001111111
	if battTempSign == 1 {
		battTemp = -battTemp
	}

	readings["BattDegC"] = int32(battTemp)

	ctlTempSign := (int32(tempReading) & 0b1000000000000000) >> 15
	ctlTemp := (int16(tempReading) & 0b0111111100000000) >> 8
	if ctlTempSign == 1 {
		ctlTemp = -ctlTemp
	}
	readings["ControllerDegC"] = int32(ctlTemp)

	return readings, nil
}

func (r *Renogy) addReading(client modbus.Client, register uint16, precision uint, reading string) {
	value, err := r.readRegister(client, register, precision)
	if err != nil {
		r.logger.Errorf("error getting reading: %s : %v", reading, err)
	} else {
		readings[reading] = value
	}
}

func (r *Renogy) readRegister(client modbus.Client, register uint16, precision uint) (result float32, err error) {
	r.mu.Lock()
	b, err := client.ReadHoldingRegisters(register, 1)
	r.mu.Unlock()
	fmt.Println(b)
	if err != nil {
		return 0, err
	}
	if len(b) > 0 {
		result = float32FromBytes(b, precision)
	} else {
		result = 0
	}
	return result, nil
}

func float32FromBytes(bytes []byte, precision uint) float32 {
	i := binary.BigEndian.Uint16(bytes)
	ratio := math.Pow(10, float64(precision))
	return float32(float64(i) / ratio)
}

// Close closes the renogy modbus.
func (r *Renogy) Close(ctx context.Context) error {
	r.mu.Lock()
	if r.handler != nil {
		err := r.handler.Close()
		if err != nil {
			r.mu.Unlock()
			return err
		}
	}
	r.mu.Unlock()
	return nil
}
