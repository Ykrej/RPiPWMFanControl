package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	rpio "github.com/stianeikeland/go-rpio/v4"
)

const APPLICATION_NAME = "RPiPWMFanControl"

var Version string

const POLLING_SPEED_SECONDS = 1
const CPU_TEMP_FILE = "/sys/class/thermal/thermal_zone0/temp"

type Config struct {
	gpioPin                 uint8
	controlFrequencyHz      uint32
	pollingRateMilliseconds uint32
	minFanSpeedPercent      uint8
	startTempCelsius        float32
	stopTempCelsius         float32
	maxTempCelsius          float32
}

func (c *Config) GetPollingRateDuration() time.Duration {
	return time.Duration(c.pollingRateMilliseconds * uint32(time.Millisecond))
}

func (c *Config) Validate() error {
	if c.startTempCelsius < 0 {
		return errors.New("startTemp must be positive")
	}

	if c.stopTempCelsius < 0 {
		return errors.New("stopTemp must be positive")
	}

	if c.maxTempCelsius < 0 {
		return errors.New("maxTemp must be positive")
	}

	if c.startTempCelsius < c.stopTempCelsius {
		return errors.New("startTemp must be >= stopTemp")
	}

	if c.startTempCelsius > c.maxTempCelsius {
		return errors.New("startTemp must be <= maxTemp")
	}

	if c.minFanSpeedPercent > 100 {
		return errors.New("minFanSpeed must be >= 0 and <= 100")
	}

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf(`Config
	GPIO Pin:          %v
	Control Frequency: %v hz
	Minimum Fan Speed: %v%%
	Start Temp:        %v °C
	Stop Temp:         %v °C
	Max Temp:          %v °C
	Polling Rate:      %v ms`,
		c.gpioPin,
		c.controlFrequencyHz,
		c.minFanSpeedPercent,
		c.startTempCelsius,
		c.stopTempCelsius,
		c.maxTempCelsius,
		c.pollingRateMilliseconds,
	)
}

func main() {
	config := loadConfigFromFlags()
	log.Println(fmt.Sprintf("%v %v", APPLICATION_NAME, Version))
	log.Println(config.String())

	err := config.Validate()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	pollingRateDuration := config.GetPollingRateDuration()

	err = rpio.Open()
	if err != nil {
		log.Fatalf("Failed to open memory range in /dev/mem: %v", err)
	}

	pin := initPwmPin(config.gpioPin, config.controlFrequencyHz)

	var cpuTemp float32
	var fanSpeedPercent uint8 = 0
	for {
		cpuTemp, err = getCpuTempCelsius()
		if err != nil {
			log.Fatalf("Failed to get cpu temp: %v", err)
		}

		desiredFanSpeedPercent := getDesiredFanSpeedPercent(
			config.startTempCelsius,
			config.stopTempCelsius,
			config.maxTempCelsius,
			cpuTemp,
			float32(fanSpeedPercent),
		)
		if desiredFanSpeedPercent != 0 && desiredFanSpeedPercent != math.MaxUint8 {
			desiredFanSpeedPercent = maxUint8(
				desiredFanSpeedPercent,
				config.minFanSpeedPercent,
			)
		}

		if desiredFanSpeedPercent != math.MaxUint8 { // uint8 max represent maintain current fan speed
			setFanSpeed(pin, desiredFanSpeedPercent)
			fanSpeedPercent = desiredFanSpeedPercent
		}
		log.Printf("CPU Temp: %v°C\tFan Speed: %v%%\n", cpuTemp, fanSpeedPercent)
		time.Sleep(pollingRateDuration)
	}
}

func loadConfigFromFlags() Config {
	gpioPin := flag.Uint("gpio", 18, "GPIO pin number for PWM control")
	controlFrequencyHz := flag.Uint("freq", 25000, "PWM control frequency in Hz")
	pollingRateMilliseconds := flag.Uint("poll", 500, "Polling rate in milliseconds")
	minFanSpeedPercent := flag.Uint("min-speed", 30, "Minimum fan speed as a percent from 0 to 100")
	startTempCelsius := flag.Float64("start-temp", 40, "Temperature (°C) to start fan")
	stopTempCelsius := flag.Float64("stop-temp", 35, "Temperature (°C) to stop fan")
	maxTempCelsius := flag.Float64("max-temp", 55, "Temperature (°C) for max fan speed")
	version := flag.Bool("version", false, "List version info and exit")
	flag.Parse()

	if *version {
		if len(Version) == 0 {
			fmt.Println("Uh oh, no version info found.")
		} else {
			fmt.Println(Version)
		}
		os.Exit(0)
	}

	return Config{
		gpioPin:                 uint8(*gpioPin),
		controlFrequencyHz:      uint32(*controlFrequencyHz),
		pollingRateMilliseconds: uint32(*pollingRateMilliseconds),
		minFanSpeedPercent:      uint8(*minFanSpeedPercent),
		startTempCelsius:        float32(*startTempCelsius),
		stopTempCelsius:         float32(*stopTempCelsius),
		maxTempCelsius:          float32(*maxTempCelsius),
	}
}

func initPwmPin(pinNum uint8, frequency uint32) rpio.Pin {
	pin := rpio.Pin(pinNum)
	pin.Mode(rpio.Pwm)
	pin.Pwm()
	pin.Freq(int(frequency))
	rpio.StartPwm()
	return pin
}

func setFanSpeed(pin rpio.Pin, percent uint8) {
	if percent > 100 {
		percent = 100
	}

	pin.DutyCycleWithPwmMode(
		uint32(percent),
		100,
		true,
	)
}

func getCpuTempCelsius() (float32, error) {
	data, err := os.ReadFile(CPU_TEMP_FILE)
	if err != nil {
		return 0, err
	}

	millicelsius, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return float32(millicelsius) / 1000, nil
}

func getDesiredFanSpeedPercent(
	startTempCelsius float32,
	stopTempCelsius float32,
	maxTempCelsius float32,
	currentTempCelsius float32,
	currentFanSpeedPercent float32,
) uint8 {
	if currentTempCelsius >= maxTempCelsius {
		return percentOfRange(stopTempCelsius, maxTempCelsius, currentTempCelsius)
	}

	if currentFanSpeedPercent > 0 { // Fan already running
		if currentTempCelsius > stopTempCelsius {
			return percentOfRange(stopTempCelsius, maxTempCelsius, currentTempCelsius)
		}
	} else { // Fan not currently running
		if currentTempCelsius >= startTempCelsius {
			return percentOfRange(stopTempCelsius, maxTempCelsius, currentTempCelsius)
		}
	}

	return 0
}

func percentOfRange(min float32, max float32, value float32) uint8 {
	// Gets the percent of the range the value represents
	if value > max {
		return 100
	}
	if value <= min {
		return 0
	}

	percent := uint8((value - min) / (max - min) * 100)
	if percent > 100 {
		percent = 100
	}
	return percent
}

func maxUint8(values ...uint8) uint8 {
	max := values[0]
	for i := 1; i < len(values); i++ {
		if max == math.MaxUint8 {
			return max
		}
		if values[i] > max {
			max = values[i]
		}
	}
	return max
}
