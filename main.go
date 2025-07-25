package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	rpio "github.com/stianeikeland/go-rpio/v4"
)


const POLLING_SPEED_SECONDS = 1
const CPU_TEMP_FILE = "/sys/class/thermal/thermal_zone0/temp"



type Config struct {
	gpioPin uint8
	controlFrequencyHz uint32
	pollingRateMilliseconds uint32
	startTempCelsius float32
	stopTempCelsius float32
	maxTempCelsius float32
}

func (c *Config) GetPollingRateDuration() time.Duration {
	return time.Duration(c.pollingRateMilliseconds * uint32(time.Millisecond))
}


func main() {
	fmt.Println("Hello World")

	config := Config{
		gpioPin: 18,
		controlFrequencyHz: 25000,
		pollingRateMilliseconds: 500,
		startTempCelsius: 45,
		stopTempCelsius: 40,
		maxTempCelsius: 65,
	}
	pollingRateDuration := config.GetPollingRateDuration()

	err := rpio.Open()
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
		
		fmt.Printf("CPU Temp: %v°C\tDesired Fan Speed: %v\tCurrent Fan Speed: %v\n", cpuTemp, desiredFanSpeedPercent, fanSpeedPercent)
		if desiredFanSpeedPercent != math.MaxUint8 {  // uint8 max represent maintain current fan speed
			setFanSpeed(pin, desiredFanSpeedPercent)
			fanSpeedPercent = desiredFanSpeedPercent
		}
		
		time.Sleep(pollingRateDuration)
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
	if (percent > 100) {
		percent = 100
	}

	fmt.Printf("Setting fan speed to %v\n", percent)

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
	// TODO: Replace 100 values with percent fan speed interpolated between stop and max temp
	if currentTempCelsius >= maxTempCelsius {
		return percentOfRange(stopTempCelsius, maxTempCelsius, currentTempCelsius)
	}

	if currentFanSpeedPercent > 0 {  // Fan already running
		if currentTempCelsius > stopTempCelsius {
			return percentOfRange(stopTempCelsius, maxTempCelsius, currentTempCelsius)
		}
	} else {  // Fan not currently running
		if currentTempCelsius >= startTempCelsius {
			return percentOfRange(stopTempCelsius, maxTempCelsius, currentTempCelsius)
		}
	}
	
	return 0
}

func percentOfRange(min float32, max float32, value float32) uint8 {
	// Gets the percent of the range the value represen
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
