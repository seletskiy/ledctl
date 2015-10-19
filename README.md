ledctl is simple tool which provides interface for controlling keyboard LEDs.

The difference between `ledctl` and `xset led` and `setleds`, that `ledctl`
can be used in "streaming" mode by reading LED control commands from stdin.

It is useful for blinking keyboard LEDs at fast rate, which is impossible with
exec'ing `xset` and `setleds` just because `exec` took too long:

```
while :; do
    echo +caps +scroll
    sleep 0.015
    echo -all
    sleep 0.015
done | sudo ledctl -Si
```

# Installation

```
go get github.com/seletskiy/ledctl
```

# Usage

See `ledctl -h` for the guide.
