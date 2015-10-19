package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/docopt/docopt-go"
)

// #include <linux/kd.h>
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
    -t <tty>   TTY name [default: /dev/tty1].
`

var leds = map[string]byte{
	"scroll": C.LED_SCR,
	"num":    C.LED_NUM,
	"caps":   C.LED_CAP,
	"all":    0xff,
}

func main() {
	args, err := docopt.Parse(
		strings.Replace(usage, "$0", os.Args[0], -1),
		nil, true, "ledctl 1", false,
	)
	if err != nil {
		panic(err)
	}

	ttyName := args["-t"].(string)

	tty, err := os.Open(ttyName)
	if err != nil {
		log.Fatalf("can't open terminal: %s", ttyName)
	}

	var (
		wg      = sync.WaitGroup{}
		done    = make(chan struct{}, 0)
		control = make(chan string, 0)
		state   = byte(0)
	)
	wg.Add(1)
	go func() {
		state, err = getLEDs(tty)
		if err != nil {
			log.Fatalf("can't get active LEDs: %s", err)
		}

		for {
			select {
			case <-done:
				wg.Done()
				return

			case command := <-control:
				state, err = applyLEDCommand(tty, command, state)
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

func getLEDs(tty *os.File) (byte, error) {
	var leds byte

	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL, tty.Fd(), uintptr(C.KDGETLED),
		uintptr(unsafe.Pointer(&leds)),
	)

	if err != 0 {
		return 0, fmt.Errorf("KDGETLED syscall error: %s", err)
	}

	return leds, nil
}

func setLEDs(tty *os.File, leds byte) error {
	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL, tty.Fd(), uintptr(C.KDSETLED),
		uintptr(leds),
	)

	if err != 0 {
		return fmt.Errorf("KDSETLED syscall error: %s", err)
	}

	return nil
}

func applyLEDCommand(tty *os.File, command string, state byte) (byte, error) {
	if len(command) < 2 {
		return state, fmt.Errorf("invalid command: %s", command)
	}

	if command[0] != '+' && command[0] != '-' {
		return state, fmt.Errorf(
			"command do not have prefix '-' or '+': %s", command,
		)
	}

	if _, ok := leds[command[1:]]; !ok {
		return state, fmt.Errorf("unknown LED name: %s", command[1:])
	}

	LEDIndex := leds[command[1:]]

	newLEDs := byte(0)
	switch command[0] {
	case '-':
		newLEDs = state & (0xff ^ LEDIndex)
	case '+':
		newLEDs = state | LEDIndex
	}

	err := setLEDs(tty, newLEDs)
	if err != nil {
		return state, fmt.Errorf("can't set LEDs: %s", err)
	}

	return newLEDs, nil
}
