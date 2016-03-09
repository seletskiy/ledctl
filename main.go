package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/docopt/docopt-go"
)

// #cgo LDFLAGS: -lX11
// #include <X11/Xlib.h>
// #include <linux/input.h>
import "C"

const usage = `ledctl - controls keyboard LEDs.

Usage:
    $0 -h | --help
    $0 [options] -S -- <command>...
    $0 [options] -Si

Options:
    -h --help  Show this help.
    -S         Set led status.
               <command> should begin either:
                 * with '+' for turning LED on;
                 * with '-' for turning LED off.
               The rest is name of the LED, possible values are:
                 * scroll;
                 * num;
                 * caps;
                 * all;
      -i       Read <command> from stdin.
`

var leds = map[string]C.int{
	"scroll": C.LED_SCROLLL,
	"num":    C.LED_NUML,
	"caps":   C.LED_CAPSL,
	"all":    -1,
}

const (
	ledOn  = C.int(1)
	ledOff = C.int(0)
)

func main() {
	args, err := docopt.Parse(
		strings.Replace(usage, "$0", os.Args[0], -1),
		nil, true, "ledctl 1", false,
	)
	if err != nil {
		panic(err)
	}

	var (
		wg      = sync.WaitGroup{}
		done    = make(chan struct{}, 0)
		control = make(chan string, 0)
		display = C.XOpenDisplay(C.CString(os.ExpandEnv("$DISPLAY")))
	)

	if display == nil {
		log.Fatalf("can't open display")
	}

	defer C.XCloseDisplay(display)

	wg.Add(1)
	go func() {
		for {
			select {
			case <-done:
				wg.Done()
				return

			case command := <-control:
				err = applyLEDCommand(display, command)
				if err != nil {
					log.Print(err)
				}
			}
		}
	}()

	if args["-i"].(bool) {
		for {
			command := ""

			_, err := fmt.Scan(&command)
			if err == io.EOF {
				break
			}

			control <- command
		}
	} else {
		commands := args["<command>"].([]string)
		for _, command := range commands {
			control <- command
		}
	}

	done <- struct{}{}
	wg.Wait()
}

func setLEDs(display *C.Display, values C.XKeyboardControl) {
	if values.led == leds["all"] {
		C.XChangeKeyboardControl(display, C.KBLedMode, &values)
	} else {
		C.XChangeKeyboardControl(display, C.KBLed|C.KBLedMode, &values)
	}
}

func applyLEDCommand(
	display *C.Display, command string,
) error {
	if len(command) < 2 {
		return fmt.Errorf("invalid command: %s", command)
	}

	if command[0] != '+' && command[0] != '-' {
		return fmt.Errorf(
			"command do not have prefix '-' or '+': %s", command,
		)
	}

	if _, ok := leds[command[1:]]; !ok {
		return fmt.Errorf("unknown LED name: %s", command[1:])
	}

	LEDIndex := leds[command[1:]]

	switch command[0] {
	case '-':
		setLEDs(display, C.XKeyboardControl{led: LEDIndex, led_mode: ledOff})
	case '+':
		setLEDs(display, C.XKeyboardControl{led: LEDIndex, led_mode: ledOn})
	}

	return nil
}
