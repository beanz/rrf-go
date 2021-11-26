# rrf-go

RepRapFirmware API with Home Assistant MQTT discovery


# Running home assistant integration

``` shell
$ docker run mhindess/rrf2mqtt:latest ha -p <duet-password> \
      --broker tcp://<mqtt-broker-ip>:1883 \
      <hostname-of-printer/cnc>
```

# Running the mock printer for testing

``` shell
$ docker run -p 8888:8888 mhindess/rrf2mqtt:latest \
      mock --bind 0.0.0.0:8888 &
$ docker run mhindess/rrf2mqtt:latest ha -p reprap \
      --broker tcp://<mqtt-broker-ip>:1883 \
      <host-ip>:8888
```

# Help for commands and options

``` shell
$ docker run mhindess/rrf2mqtt:latest --help
$ docker run mhindess/rrf2mqtt:latest ha --help
$ docker run mhindess/rrf2mqtt:latest mock --help
```
