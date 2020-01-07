/**
This source file defines the config options.
*/
package main

import (
	"io"

	"github.com/BurntSushi/toml"
	//"fmt"
)

type SyncOption struct {
	// Sync Options ----------------------------------
	Enabled     bool
	Output      string
	IndexName   string
	ReSegment   bool
	RemoveOld   bool
	CleanFolder bool
}

type RecordOption struct {
	// Record Options --------------------------------
	Enabled           bool
	Output            string
	SegmentRewrite    string
	Reindex           bool
	ReindexFormat     string
	ReindexBy         string // hour/minute
	Timeshifting      bool
	TimeshiftFilename string
	TimeshiftDuration int
}

type SourceOption struct {
	Urls []string
}

type HttpOption struct {
	Enabled       bool
	Listen        string // eg:  tcp://0.0.0.0:8080  or  unix:///tmp/test.sock
	Days          int    // Max shifting days.
	Max           int    // Max length of playlist in minutes.
	SegmentPrefix string // Segment prefix when generating playlist.
	CacheNum      int    // Num of Cache entries for avoid re-generating playlist.
	CacheValid    int    // Cache valid duration in seconds.
}

type Option struct {
	// Global Options --------------------------------
	LogFile           string
	LogLevel          string
	Timeout           int
	Retries           int
	UserAgent         string
	MaxSegments       int
	TimestampType     string // local|program|segment
	TimestampFormat   string
	TimezoneShift     int
	TargetDuration    int
	ProgramTimeFormat string
	ProgramTimezone   string

	// Sync Option
	Sync SyncOption
	// Record Option
	Record RecordOption
	// Source URLs.
	Source SourceOption
	// Http Service
	Http HttpOption
}

func CheckConfiguration(option *Option, output io.Writer) {
	var _print = func(s string) {
		io.WriteString(output, s)
	}
	_print("Checking options ...\n")
	if nil == option {
		_print("Invalid configuration!!!\n")
		return
	}
	_print("\n")
	toml.NewEncoder(output).Encode(option)
	//_print(fmt.Sprintf("LogFile: %s\n", option.LogFile))
	//_print(fmt.Sprintf("LogLevel: %s\n", option.LogLevel))
	//_print(fmt.Sprintf("Timeout: %d\n", option.Timeout))
	//_print(fmt.Sprintf("Retries: %d\n", option.Retries))
	//_print(fmt.Sprintf("UserAgent: %s\n", option.UserAgent))
	//_print(fmt.Sprintf("MaxSegments: %d\n", option.MaxSegments))
	//_print(fmt.Sprintf("TimestampType: %s\n", option.TimestampType))
	//_print(fmt.Sprintf("TimestampFormat: %s\n", option.TimestampFormat))
	//_print(fmt.Sprintf("TimezoneShift: %d\n", option.TimezoneShift))
	//_print(fmt.Sprintf("TargetDuration: %d\n", option.TargetDuration))
	//_print(fmt.Sprintf("ProgramTimeFormat: %s\n", option.ProgramTimeFormat))
	//_print(fmt.Sprintf("ProgramTimezone: %s\n", option.ProgramTimezone))
	//
	//_print("\nSync Options:\n")
	//_print(fmt.Sprintf("  Enabled: %t\n", option.Sync.Enabled))
	//_print(fmt.Sprintf("  Output: %s\n", option.Sync.Output))
	//_print(fmt.Sprintf("  IndexName: %s\n", option.Sync.IndexName))
	//_print(fmt.Sprintf("  RemoveOld: %t\n", option.Sync.RemoveOld))
	//_print(fmt.Sprintf("  CleanFolder: %t\n", option.Sync.CleanFolder))
	//
	//_print("\nRecord Options:\n")
	//_print(fmt.Sprintf("  Enabled: %t\n", option.Record.Enabled))
	//_print(fmt.Sprintf("  Output: %s\n", option.Record.Output))
	//_print(fmt.Sprintf("  SegmentRewrite: %s\n", option.Record.SegmentRewrite))
	//_print(fmt.Sprintf("  Reindex: %t\n", option.Record.Reindex))
	//_print(fmt.Sprintf("  ReindexBy: %s\n", option.Record.ReindexBy))
	//_print(fmt.Sprintf("  ReindexFormat: %s\n", option.Record.ReindexFormat))
	//
	//_print("\nHTTP Options:\n")
	//_print(fmt.Sprintf("  Enabled: %t\n", option.Http.Enabled))
	//_print(fmt.Sprintf("  Listen: %s\n", option.Http.Listen))
	//_print(fmt.Sprintf("  Days: %d\n", option.Http.Days))
	//_print(fmt.Sprintf("  Max: %d\n", option.Http.Max))
	//_print(fmt.Sprintf("  SegmentPrefix: %s\n", option.Http.SegmentPrefix))
	//_print(fmt.Sprintf("  CacheNum: %d\n", option.Http.CacheNum))
	//_print(fmt.Sprintf("  CacheValid: %d\n", option.Http.CacheValid))
	_print("\n")
	_print("Configuration validated!\n")
}

func LoadConfiguration(filename string, option *Option) (e error) {
	_, e = toml.DecodeFile(filename, option)
	return e
}
