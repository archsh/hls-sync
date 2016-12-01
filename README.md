# hls-sync

Sync hls live stream to local disk.

## Scenarios:

   * Sync live hls streams from remote hls server.
   * Record live streams to local disks.

## Usage:

    hls-sync [OPTIONS,...] SOURCE_URL1 ...

You can run with several URL as failover mechanism. `hls-sync` the first one and than the second one if there's a failure.

### Options:
#### Control Options
  - `c` string
        Configuration file instead of command line parameters. Default empty means using parameters.
  - `C ` 
        Check options.
  - `v` 
        Show version info.
       
#### Global Options

  - `L` string
        Logging output file. Default 'stdout'.
  - `MS` int
        Max segments in playlist. (default 20)
  - `R` int
          Retries. (default 1)
  - `T` int
        Request timeout.  (default 5)
  - `TD` int
        Target duration of source. Real target duration will be used when set to 0.
  - `TS` int
        Timezone shifting by minutes when timestamp is not matching local timezone.
  - `TT` string
        Timestamp type: local, program, segment. (default "program")
  - `UA` string
        User Agent.  (default "hls-sync v0.2.0")
  - `V` string
        Logging level.  (default "INFO")
  - `PF` string
        To fit some stupid encoders which generated stupid time format. (default "2006-01-02T15:04:05.999999999Z07:00")
        eg: set it to "2006-01-02T15:04:05z" if you have something like: "#EXT-X-PROGRAM-DATE-TIME:2016-11-29T17:55:02z"
  - `PZ` string
        Timezone for PROGRAM-DATE-TIME. Default is 'UTC'.

#### Sync Options
  - `S`
        Sync enabled.
  - `SO` string
        A base path for synced segments and play list. (default ".")
  - `OI` string
        Index playlist filename. (default "live.m3u8")
  - `RM`
        Remove old segments.

#### Record Options
  - `RC`
        Record enabled.
  - `RB` string
        Re-index by 'hour' or 'minute'. (default "hour")
  - `RF` string
        Re-index M3U8 filename format. (default "%Y/%m/%d/%H/index.m3u8")
  - `RI`
        Re-index playlist when recording.
  - `RO` string
        Record output path. (default ".")
  - `SR` string
        Segment filename rewrite rule. Default empty means simply copy. (default "%Y/%m/%d/%H/live-#:04.ts")
  - `TF` string
        Timestamp format when using timestamp type as 'segment'.

#### HTTP Options
Using HTTP interface to provide time shifting playlist access for some situation. To use this feature, recording and re-indexing should enabled.
  - `H`	Enable HTTP service for time shifting playlist.
  - `LS` string
    	HTTP listening address. support tcp:// or unix:// (default "unix://./hls-sync.sock")
  - `MX` int
    	Max length of playlist in hours. (default 6)
  - `SD` int
    	Max time shifting days for playlist. (default 7)
  - `SP` string
        Segment prefix when generating playlist.
  - `CN` int
        Num of Cache entries for avoid re-generating playlist. Default is 128.
  - `CV` int
        Cache valid duration in seconds. Default is 60.

**HTTP Interfaces:**

  - `GET /?playlist={start-timestamp}_{end-timestamp}.m3u8`
    eg: /?playlist=1479998100_1480004640.m3u8
  -	`GET /?start={start-timestamp}&duration={duration-in-seconds}`
    eg: /?start=1479998100&duration=6540
  - `GET /?start={start-timestamp}&end={end-timestamp}`
    eg: /?start=1479998100&end=1480004640


## Example

### Record stream:
    ./hls-sync -TT local -MS 10 -TD 5 -V DEBUG -RC -RO /tmp/channel1 -SR '%Y/%m/%d/%H/live-%04d.ts' -RI -RF '%Y/%m/%d/%H/index.m3u8' http://live1.example.com/chan01/live.m3u8 http://live2.example.com/chan01/live.m3u8

### Sync stream only:
    ./hls-sync  -TT local -MS 10 -TD 5 -V DEBUG -S -SO /tmp/channel1 -RM -OI 'live.m3u8' http://live1.example.com/chan01/live.m3u8 http://live2.example.com/chan01/live.m3u8    
    
### Configuration Example
    ./hls-sync -c hls-sync.toml.example
`hls-sync` use a TOML configuration file. Please check the following example:
```
log_file = ""
log_level = "DEBUG"
timeout = 10
retries = 3
user_agent = "HLS-SYNC"
timestamp_type="program"
timestamp_format=""
timezone_shift=0
target_duration=5
program_time_format=""
program_timezone="UTC"

[source]
urls=["http://live1.example.com/chan01/live.m3u8"]

[sync]
enabled=true
output="./"
remove_old=true

[record]
enabled=true
output="."
segment_rewrite="%Y/%m/%d/%H/live-#:04.ts"
reindex=true
reindex_by="hour"
reindex_format="%Y/%m/%d/%H/index.m3u8"

[http]
enabled=true
listen="unix://./hlssync.sock"
days=7
max=4
segment_prefix=""
cache_num=128
cache_valid=60
```

