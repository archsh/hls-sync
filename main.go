/*
   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.

*/

/**
This source file contains the entry and command line argument processing.
*/
package main

//go:generate sh ./gen_version.sh version.go

import (
    "flag"
    "fmt"
    "os"
    "time"
)

//const VERSION = "0.9.24-dev"

var logging_config = LoggingConfig{Format: DEFAULT_FORMAT, Level: "DEBUG"}

func Usage() {
    guide := `
Scenarios:
  (1) Sync live hls streams from remote hls server.
  (2) Record live streams to local disks.

Usage:
  hls-sync [OPTIONS,...] [URLs ...]

Options:
`
    os.Stdout.Write([]byte(guide))
    flag.PrintDefaults()
}

func main() {
    option := Option{}
    // Global Arguments ================================================================================================
    //LogFile string
    flag.StringVar(&option.LogFile, "L", "", "Logging output file. Default 'stdout'.")
    //LogLevel string
    flag.StringVar(&option.LogLevel, "V", "INFO", "Logging level. ")
    //Timeout int
    flag.IntVar(&option.Timeout, "T", 5, "Request timeout. ")
    //Retries int
    flag.IntVar(&option.Retries, "R", 1, "Retries.")
    //UserAgent string
    flag.StringVar(&option.UserAgent, "UA", "hls-sync "+VERSION+"("+TAG+")", "User Agent. ")
    //MaxSegments int
    flag.IntVar(&option.MaxSegments, "MS", 20, "Max segments in playlist.")
    //TimestampType string  // local|program|segment
    flag.StringVar(&option.TimestampType, "TT", "program", "Timestamp type: local, program, segment.")
    //TimestampFormat string
    flag.StringVar(&option.TimestampFormat, "TF", "", "Timestamp format when using timestamp type as 'segment'.")
    //TimezoneShift int
    flag.IntVar(&option.TimezoneShift, "TS", 0, "Timezone shifting by minutes when timestamp is not matching local timezone.")
    //TargetDuration int
    flag.IntVar(&option.TargetDuration, "TD", 0, "Target duration of source. Real target duration will be used when set to 0.")
    //ProgramTimeFormat string
    flag.StringVar(&option.ProgramTimeFormat, "PF", time.RFC3339Nano, "To fit some stupid encoders which generated stupid time format.")
    //ProgramTimezone string
    flag.StringVar(&option.ProgramTimezone, "PZ", "UTC", "Timezone for PROGRAM-DATE-TIME.")
    // Sync Arguments ==================================================================================================
    //Enabled bool
    flag.BoolVar(&option.Sync.Enabled, "S", false, "Sync enabled.")
    //Output string
    flag.StringVar(&option.Sync.Output, "SO", ".", "A base path for synced segments and play list.")
    //IndexName string
    flag.StringVar(&option.Sync.IndexName, "OI", "live.m3u8", "Index playlist filename.")
    //ReSegment bool
    flag.BoolVar(&option.Sync.ReSegment, "RS", false, "ReSegment enabled.")
    //RemoveOld bool
    flag.BoolVar(&option.Sync.RemoveOld, "RM", false, "Remove old segments.")
    //CleanFolder bool
    flag.BoolVar(&option.Sync.CleanFolder, "CF", false, "Clean target output folder.")
    // Record Arguments ================================================================================================
    //Enabled bool
    flag.BoolVar(&option.Record.Enabled, "RC", false, "Record enabled.")
    //Output string
    flag.StringVar(&option.Record.Output, "RO", ".", "Record output path.")
    //SegmentRewrite string
    flag.StringVar(&option.Record.SegmentRewrite, "SR", "%Y/%m/%d/%H/live-#:04.ts", "Segment filename rewrite rule. Default empty means simply copy.")
    //Reindex bool
    flag.BoolVar(&option.Record.Reindex, "RI", false, "Re-index playlist when recording.")
    //ReindexFormat string
    flag.StringVar(&option.Record.ReindexFormat, "RF", "%Y/%m/%d/%H/index.m3u8", "Re-index M3U8 filename format.")
    //ReindexBy string // hour/minute
    flag.StringVar(&option.Record.ReindexBy, "RB", "hour", "Re-index by 'hour' or 'minute'.")
    //Timeshifting bool
    flag.BoolVar(&option.Record.Timeshifting, "ST", false, "Enable timeshifting playlist.")
    //TimeshiftFilename string
    flag.StringVar(&option.Record.TimeshiftFilename, "SF", "timeshift.m3u8", "Timeshifting playlist filename.")
    //TimeshiftDuration int
    flag.IntVar(&option.Record.TimeshiftDuration, "SH", 3, "Timeshift duation in hour(s).")
    // HTTP Service Arguments ==========================================================================================
    // Enabled bool
    flag.BoolVar(&option.Http.Enabled, "H", false, "Enable HTTP service for playback playlist.")
    // Listen string
    flag.StringVar(&option.Http.Listen, "LS", "unix://./hls-sync.sock", "HTTP listening address. support tcp:// or unix://")
    // Days int
    flag.IntVar(&option.Http.Days, "SD", 7, "Max time playback days for playlist.")
    // Max int
    flag.IntVar(&option.Http.Max, "MX", 6, "Max length of playlist in hours.")
    // SegmentPrefix string
    flag.StringVar(&option.Http.SegmentPrefix, "SP", "", "Segment prefix when generating playlist.")
    // CacheNum int
    flag.IntVar(&option.Http.CacheNum, "CN", 128, "Num of Cache entries for avoid re-generating playlist.")
    // CacheValid int
    flag.IntVar(&option.Http.CacheValid, "CV", 60, "Cache valid duration in seconds.")
    // Functional Arguments ============================================================================================
    var config string
    flag.StringVar(&config, "c", "", "Configuration file instead of command line parameters. Default empty means using parameters.")
    var check bool
    flag.BoolVar(&check, "C", false, "Check options.")
    var showVersion bool
    flag.BoolVar(&showVersion, "v", false, "Display version info.")
    flag.Parse()

    if showVersion {
        os.Stderr.Write([]byte(fmt.Sprintf("hls-sync %v (%s) Built @ %s \n", VERSION, TAG, BUILD_TIME)))
        os.Exit(0)
    }
    os.Stderr.Write([]byte(fmt.Sprintf("hls-sync %v (%s)- HTTP Live Streaming (HLS) Synchronizer.\n", VERSION, TAG)))
    os.Stderr.Write([]byte("Copyright (C) 2015 Mingcai SHEN <archsh@gmail.com>. Licensed for use under the GNU GPL version 3.\n"))
    if config != "" {
        if e := LoadConfiguration(config, &option); e != nil {
            os.Stderr.Write([]byte(fmt.Sprintf("Load config<%s> failed: %s.\n", config, e)))
            os.Exit(1)
        } else {
            os.Stderr.Write([]byte(fmt.Sprintf("Loaded config from <%s>.\n", config)))
        }
        if flag.NArg() > 0 {
            option.Source.Urls = append(option.Source.Urls, flag.Args()...)
        }
    } else {
        if flag.NArg() < 1 && !check {
            os.Stderr.Write([]byte("\n\n!!! At least one source URL is required!\n"))
            Usage()
            os.Exit(1)
        } else {
            option.Source.Urls = flag.Args()
        }
    }
    if check {
        CheckConfiguration(&option, os.Stderr)
        os.Exit(0)
    }
    if option.Retries < 1 {
        option.Retries = 1
    }
    if option.ProgramTimeFormat == "" {
        option.ProgramTimeFormat = time.RFC3339Nano
    }

    logging_config.Filename = option.LogFile
    logging_config.Level = option.LogLevel
    if option.LogFile != "" {
        InitializeLogging(&logging_config, false, logging_config.Level)
    } else {
        InitializeLogging(&logging_config, true, logging_config.Level)
    }
    defer DeinitializeLogging()

    if sync, e := NewSynchronizer(&option); e != nil {
        os.Stderr.Write([]byte(fmt.Sprintf("Start failed: %s.\n", e)))
        os.Exit(1)
    } else {
        sync.Run()
    }
}
