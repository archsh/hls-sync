/**
	This source file defines the config options.
 */
package main

import (
    "github.com/BurntSushi/toml"
    "io"
    //"fmt"
)

type SyncOption struct {
    // Sync Options ----------------------------------
    Enabled      bool
    Output       string
    Index_Name   string
    Remove_Old   bool
    Clean_Folder bool
}

type RecordOption struct {
                              // Record Options --------------------------------
    Enabled            bool
    Output             string
    Segment_Rewrite    string
    Reindex            bool
    Reindex_Format     string
    Reindex_By         string // hour/minute
    Timeshifting       bool
    Timeshift_filename string
    Timeshift_duration int
}

type SourceOption struct {
    Urls []string
}

type HttpOption struct {
    Enabled        bool
    Listen         string // eg:  tcp://0.0.0.0:8080  or  unix:///tmp/test.sock
    Days           int    // Max shifting days.
    Max            int    // Max length of playlist in minutes.
    Segment_Prefix string // Segment prefix when generating playlist.
    Cache_Num      int    // Num of Cache entries for avoid re-generating playlist.
    Cache_Valid    int    // Cache valid duration in seconds.
}

type Option struct {
                               // Global Options --------------------------------
    Log_File            string
    Log_Level           string
    Timeout             int
    Retries             int
    User_Agent          string
    Max_Segments        int
    Timestamp_type      string // local|program|segment
    Timestamp_Format    string
    Timezone_shift      int
    Target_Duration     int
    Program_Time_Format string
    Program_Timezone    string
                               // Sync Option
    Sync                SyncOption
                               // Record Option
    Record              RecordOption
                               // Source URLs.
    Source              SourceOption
                               // Http Service
    Http                HttpOption
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
    //_print(fmt.Sprintf("Log_File: %s\n", option.Log_File))
    //_print(fmt.Sprintf("Log_Level: %s\n", option.Log_Level))
    //_print(fmt.Sprintf("Timeout: %d\n", option.Timeout))
    //_print(fmt.Sprintf("Retries: %d\n", option.Retries))
    //_print(fmt.Sprintf("User_Agent: %s\n", option.User_Agent))
    //_print(fmt.Sprintf("Max_Segments: %d\n", option.Max_Segments))
    //_print(fmt.Sprintf("Timestamp_type: %s\n", option.Timestamp_type))
    //_print(fmt.Sprintf("Timestamp_Format: %s\n", option.Timestamp_Format))
    //_print(fmt.Sprintf("Timezone_shift: %d\n", option.Timezone_shift))
    //_print(fmt.Sprintf("Target_Duration: %d\n", option.Target_Duration))
    //_print(fmt.Sprintf("Program_Time_Format: %s\n", option.Program_Time_Format))
    //_print(fmt.Sprintf("Program_Timezone: %s\n", option.Program_Timezone))
    //
    //_print("\nSync Options:\n")
    //_print(fmt.Sprintf("  Enabled: %t\n", option.Sync.Enabled))
    //_print(fmt.Sprintf("  Output: %s\n", option.Sync.Output))
    //_print(fmt.Sprintf("  Index_Name: %s\n", option.Sync.Index_Name))
    //_print(fmt.Sprintf("  Remove_Old: %t\n", option.Sync.Remove_Old))
    //_print(fmt.Sprintf("  Clean_Folder: %t\n", option.Sync.Clean_Folder))
    //
    //_print("\nRecord Options:\n")
    //_print(fmt.Sprintf("  Enabled: %t\n", option.Record.Enabled))
    //_print(fmt.Sprintf("  Output: %s\n", option.Record.Output))
    //_print(fmt.Sprintf("  Segment_Rewrite: %s\n", option.Record.Segment_Rewrite))
    //_print(fmt.Sprintf("  Reindex: %t\n", option.Record.Reindex))
    //_print(fmt.Sprintf("  Reindex_By: %s\n", option.Record.Reindex_By))
    //_print(fmt.Sprintf("  Reindex_Format: %s\n", option.Record.Reindex_Format))
    //
    //_print("\nHTTP Options:\n")
    //_print(fmt.Sprintf("  Enabled: %t\n", option.Http.Enabled))
    //_print(fmt.Sprintf("  Listen: %s\n", option.Http.Listen))
    //_print(fmt.Sprintf("  Days: %d\n", option.Http.Days))
    //_print(fmt.Sprintf("  Max: %d\n", option.Http.Max))
    //_print(fmt.Sprintf("  Segment_Prefix: %s\n", option.Http.Segment_Prefix))
    //_print(fmt.Sprintf("  Cache_Num: %d\n", option.Http.Cache_Num))
    //_print(fmt.Sprintf("  Cache_Valid: %d\n", option.Http.Cache_Valid))
    _print("\n")
    _print("Configration validated!\n")
}

func LoadConfiguration(filename string, option *Option) (e error) {
    _, e = toml.DecodeFile(filename, option)
    return e
}

