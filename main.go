package main

import (
	"fmt"
	"log"
	"time"

	rpio "github.com/stianeikeland/go-rpio/v4"
)


const POLLING_SPEED_SECONDS = 1
const CPU_TEMP_FILE = "/sys/class/thermal/thermal_zone/temp"



type Config struct {
	gpioPin uint8
	controlFrequencyHz uint32
	pollingRateMilliseconds uint32
}

func (c *Config) GetPollingRateDuration() time.Duration {
	return time.Duration(c.pollingRateMilliseconds * uint32(time.Millisecond))
}


func main() {
	fmt.Println("Hello World")

	config := Config{
		18,
		25000,
		500,
	}
	pollingRateDuration := config.GetPollingRateDuration()

	err := rpio.Open()
	if err != nil {
		log.Fatalf("Failed to open memory range in /dev/mem, %v", err)
	}

	pin := initPwmPin(config.gpioPin, config.controlFrequencyHz)

	var i uint8 = 0
	for {
		setFanSpeed(pin, i)
		i += 1
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

	fmt.Printf("Setting fan speed to %v%\n", percent)

	pin.DutyCycleWithPwmMode(
		uint32(percent),
		100,
		true,
	)
}
