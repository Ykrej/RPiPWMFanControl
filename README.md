# RPiPWMFanControl
Fan control based on cpu temp for raspberry pi using hardware PWM.

## Setup/Install
Download the `rpi-pwm-fancontrol` binary from [releases](https://github.com/Ykrej/RPiPWMFanControl/releases) or build the program yourself with the `build.sh` script.

NOTE: Running the program as non-root silently fails to control the pwm signal. Please open an issue if you find a solution to this.

### Configuration
All configuration is done through the command line interface.

The default `--freq` value works for the Noctua NF-A4x10 5v PWM fan but is likely to require a different value for different fans/brands.

This set of options could be out of date. Check the output from `rpi-pwm-fancontrol --help` for up to date options.
``` 
Usage of /usr/local/bin/rpi-pwm-fancontrol:
  -freq uint
        PWM control frequency in Hz (default 25000)
  -gpio uint
        GPIO pin number for PWM control (default 18)
  -max-temp float
        Temperature (°C) for max fan speed (default 55)
  -min-speed uint
        Minimum fan speed as a percent from 0 to 100 (default 30)
  -poll uint
        Polling rate in milliseconds (default 500)
  -start-temp float
        Temperature (°C) to start fan (default 40)
  -stop-temp float
        Temperature (°C) to stop fan (default 35)
  -version
        List version info and exit
```

### Systemd/Autostart
To get the program to run on startup I use systemd.
1. Copy the binary to `/usr/local/bin`
2. Create the following systemd service at `/etc/systemd/system/rpi-pwm-fancontrol.service`. Edit the configuration on the ExecStart value as you see fit.
```ini
[Unit]
Description=controls the pwm fan based on cpu temp

[Service]
Type=simple
User=root
Restart=always
RestartSec=3
ExecStart=/usr/local/bin/rpi-pwm-fancontrol

[Install]
WantedBy=multi-user.target
```

3. Start the service
```bash
sudo systemctl daemon-reload
sudo systemctl enable rpi-pwm-fancontrol.service
sudo systemctl start rpi-pwm-fancontrol.service
```
