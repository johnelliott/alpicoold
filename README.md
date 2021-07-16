# demo
```bash
sudo btmon
sudo bluetoothctl
sudo LOGLEVEL=debug go run . -timeout 30s -pollrate 1s|jq -c
sudo LOGLEVEL=panic go run . -timeout 1m -pollrate 1s|jq .Temp
```

# links
<https://pkg.go.dev/github.com/muka/go-bluetooth@v0.0.0-20210508070623-03c23c62f181>

# more
```
sudo dbus-monitor --system "type=error"
```

"fefe1102000101022444fc0400010000fb000477"
unlock, 36 deg, gt


Here's a report from the device:
```
0000   fe fe 15 01 01 01 01 02 44 44 fc 04 00 01 00 00
0010   fb 00 45 64 0d 02 05 53
```



Sun 2021-07-04 19:27
turning on the compressor by opening, then just watching traffic

0000   fe fe 15 01 01 01 01 02 44 44 fc 04 00 01 00 00
0010   fb 00 43 64 0d 02 05 51
             ^ this is the temp, in farenheit degrees in ex, so 43 is 67F
and the temp is the 11th byte of the notification payload


