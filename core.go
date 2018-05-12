/**
This source file contains the playlist updating and segments downloading.
*/
package main

import (
    "bytes"
    "errors"
    "io/ioutil"
    "net/http"
    "strings"
    "sync"
    "time"

    log "github.com/Sirupsen/logrus"
    "github.com/archsh/go.m3u8"
    "github.com/golang/groupcache/lru"
    //"net/url"
    "fmt"
    "os"

    "github.com/archsh/go.timefmt"
)

type Synchronizer struct {
    option           *Option
    client           *http.Client
    program_timezone *time.Location
    httpCache        *lru.Cache
    sourceCrc16      string
}

type SegmentMessage struct {
    _type            SyncType
    _hit             bool
    _target_duration float64
    playlist         *m3u8.MediaPlaylist
    segment          *m3u8.MediaSegment
    response         *http.Response
}

func NewSynchronizer(option *Option) (s *Synchronizer, e error) {
    if len(option.Source.Urls) < 1 {
        return nil, errors.New("\n\n!!! At least one source URL is required!\n\n")
    }
    s = new(Synchronizer)
    s.option = option
    s.client = &http.Client{Timeout: time.Duration(option.Timeout) * time.Second}
    s.sourceCrc16 = fmt.Sprintf("%04x", CRC16([]byte(option.Source.Urls[0])))
    if s.program_timezone, e = time.LoadLocation(option.Program_Timezone); nil != e {
        return nil, e
    //} else {
    //    m3u8.ProgramTimeLocation = s.program_timezone
    //}
    //if option.Program_Time_Format != "" {
    //    m3u8.ProgramTimeFormat = option.Program_Time_Format
    }
    return s, nil
}

func (self *Synchronizer) Run() {
    log.Infoln("Synchronizer.Run > Starting hls-sync ...")
    syncChan := make(chan *SyncMessage, 20)
    recordChan := make(chan *RecordMessage, 20)
    segmentChan := make(chan *SegmentMessage, 20)
    //m3u8.ProgramTimeFormat = self.option.Program_Time_Format
    //m3u8.ProgramTimeLocation = self.program_timezone
    if self.option.Http.Enabled {
        if !self.option.Record.Enabled || !self.option.Record.Reindex {
            os.Stderr.Write([]byte("\n\n!!! Record(-RC) and Re-index(-RI) should enabled to enable HTTP service !\n"))
            os.Exit(1)
        }
    }
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        self.playlistProc(segmentChan)
        wg.Done()
    }()
    wg.Add(1)
    go func() {
        self.segmentProc(segmentChan, syncChan, recordChan)
        wg.Done()
    }()
    if self.option.Sync.Enabled {
        wg.Add(1)
        go func() {
            self.syncProc(syncChan)
            wg.Done()
        }()
    }
    if self.option.Record.Enabled {
        wg.Add(1)
        go func() {
            self.recordProc(recordChan)
            wg.Done()
        }()
    }
    if self.option.Http.Enabled {
        wg.Add(1)
        go func() {
            self.HttpServe()
            wg.Done()
        }()
    }
    wg.Wait()
}

func (self *Synchronizer) playlistProc(segmentChan chan *SegmentMessage) {
    cache := lru.New(self.option.Max_Segments)
    retry := 0
    timezone_shift := time.Minute * time.Duration(self.option.Timezone_shift)
    timestamp_type := TST_LOCAL
    switch strings.ToLower(self.option.Timestamp_type) {
    case "local":
        timestamp_type = TST_LOCAL
    case "segment":
        timestamp_type = TST_SEGMENT
    default:
        timestamp_type = TST_PROGRAM
    }
    src_idx := 0
    last_new_segment := time.Now()
    for {
        if retry >= self.option.Retries {
            if len(self.option.Source.Urls) > (src_idx + 1) {
                src_idx += 1
                retry = 0
            } else if src_idx > 0 {
                src_idx = 0
                retry = 0
            }
        }
        urlStr := self.option.Source.Urls[src_idx]
        req, err := http.NewRequest("GET", urlStr, nil)
        if err != nil {
            log.Errorln("Create Request failed:>", err)
            continue
        }
        resp, err := self.doRequest(req)
        if err != nil {
            log.Errorln("doRequest failed:> ", retry, err)
            time.Sleep(time.Duration(1) * time.Second)
            retry++
            continue
        }
        respBody, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            log.Errorln("Read Playlist Response body failed:> ", retry, err)
            time.Sleep(time.Duration(1) * time.Second)
            retry++
            continue
        }
        buffer := bytes.NewBuffer(respBody)
        playlist, listType, err := m3u8.Decode(*buffer, true, self.option.Program_Time_Format, self.program_timezone)
        if err != nil {
            log.Errorln("Decode playlist failed:> ", retry, err)
            time.Sleep(time.Duration(1) * time.Second)
            retry++
            continue
        }
        resp.Body.Close()
        mpl_updated := false
        lastTimestamp := time.Now()
        seg_num := 0
        if listType == m3u8.MEDIA {
            mpl := playlist.(*m3u8.MediaPlaylist)
            retry = 0
            for _, v := range mpl.Segments {
                if v != nil {
                    //log.Debugln("Segment:> ", v.URI, v.ProgramDateTime)
                    seg_num++
                    t, hit := cache.Get(v.URI)
                    if !hit {
                        if timestamp_type == TST_SEGMENT {
                            v.ProgramDateTime, _ = timefmt.Strptime(v.URI, self.option.Timestamp_Format)
                        }
                        if timestamp_type == TST_LOCAL || v.ProgramDateTime.Year() < 2016 || v.ProgramDateTime.Month() == 0 || v.ProgramDateTime.Day() == 0 {
                            v.ProgramDateTime = lastTimestamp
                            lastTimestamp = lastTimestamp.Add(time.Duration(v.Duration*1000) * time.Millisecond)
                        } else {
                            v.ProgramDateTime = v.ProgramDateTime.Add(timezone_shift)
                        }
                        cache.Add(v.URI, v.ProgramDateTime)
                        last_new_segment = time.Now()
                        log.Infof("New segment:> %d | %s | %f | %s \n", mpl.SeqNo, v.URI, v.Duration, v.ProgramDateTime)
                        if self.option.Sync.Enabled || self.option.Record.Enabled {
                            // Only get segments when sync or record enabled.
                            msg := &SegmentMessage{}
                            msg._type = SEGMEMT
                            msg._hit = false
                            msg._target_duration = mpl.TargetDuration
                            msg.segment = v
                            msg.response = resp
                            segmentChan <- msg
                        }
                        mpl_updated = true
                    } else {
                        v.ProgramDateTime = t.(time.Time)
                        lastTimestamp = v.ProgramDateTime.Add(time.Duration(v.Duration) * time.Second)
                        if self.option.Sync.Enabled || self.option.Record.Enabled {
                            // Only get segments when sync or record enabled.
                            msg := &SegmentMessage{}
                            msg._type = SEGMEMT
                            msg._hit = true
                            msg._target_duration = mpl.TargetDuration
                            msg.segment = v
                            msg.response = resp
                            segmentChan <- msg
                        }
                    }
                }
            }
            if time.Now().Sub(last_new_segment) >= time.Duration(mpl.TargetDuration)*time.Second*time.Duration(seg_num) {
                log.Warningf("Long time without new segment, please check stream continuity. [ %s -> %s ] \n", last_new_segment, time.Now())
            }
            if self.option.Sync.Enabled && mpl_updated {
                msg := &SegmentMessage{}
                msg._type = PLAYLIST
                msg._target_duration = mpl.TargetDuration
                msg.segment = nil
                msg.response = resp
                msg.playlist = mpl
                segmentChan <- msg
            }
            if mpl.Closed {
                log.Errorln("Media Playlist closed ? This should not be happened!")
                //close(segmentChan)
                //return
                retry++
            } else {
                time.Sleep(time.Duration(int64((mpl.TargetDuration / 2) * 1000000000)))
            }
        } else {
            log.Errorln("> Not a valid media playlist.", retry)
            retry++
        }
    }
    // Close segment message channel.
    close(segmentChan)
}

func (self *Synchronizer) segmentProc(segmentChan chan *SegmentMessage, syncChan chan *SyncMessage, recordChan chan *RecordMessage) {
    for msg := range segmentChan {
        if nil == msg {
            continue
        }
        if msg._type == PLAYLIST {
            le_msg := &SyncMessage{}
            le_msg._type = msg._type
            le_msg.playlist = msg.playlist
            le_msg.segment = nil
            le_msg.seg_buffer = nil
            syncChan <- le_msg
        } else {
            var msURI string
            var msFilename string
            if strings.HasPrefix(msg.segment.URI, "http://") || strings.HasPrefix(msg.segment.URI, "https://") {
                //msURI, _ = url.QueryUnescape(msg.segment.URI)
                msURI = msg.segment.URI
                msFilename, _ = timefmt.Strftime(msg.segment.ProgramDateTime, self.sourceCrc16+"_%Y%m%d-%H%M%S.ts")
            } else {
                msUrl, _ := msg.response.Request.URL.Parse(msg.segment.URI)
                //msURI, _ = url.QueryUnescape(msUrl.String())
                msURI = msUrl.String()
                if self.option.Sync.Resegment {
                    msFilename, _ = timefmt.Strftime(msg.segment.ProgramDateTime, self.sourceCrc16+"_%Y%m%d-%H%M%S.ts")
                } else {
                    msFilename = msg.segment.URI
                }
                //msFilename,_ = timefmt.Strftime(msg.segment.ProgramDateTime, "%Y%m%d-%H%M%S.ts")
            }
            msg.segment.URI = msFilename
            if msg._hit {
                continue
            }
            log.Debugln("Downloading new segment:> ", msg.segment.URI)
            for i := 0; i < self.option.Retries; i++ {
                req, err := http.NewRequest("GET", msURI, nil)
                if err != nil {
                    log.Errorf("Create new request failed:> %s\n", err)
                    continue
                }
                resp, err := self.doRequest(req)
                if err != nil {
                    log.Errorf("Do request failed:> %s \n", err)
                    time.Sleep(time.Duration(1) * time.Second)
                    continue
                }
                if resp.StatusCode != 200 {
                    log.Errorf("Received HTTP %d for %s \n", resp.StatusCode, msURI)
                    time.Sleep(time.Duration(1) * time.Second)
                    continue
                }
                respBody, err := ioutil.ReadAll(resp.Body)
                if err != nil {
                    log.Errorln("Read Segment Response body failed:> ", err)
                    time.Sleep(time.Duration(1) * time.Second)
                    continue
                }
                resp.Body.Close()
                buffer := bytes.NewBuffer(respBody)
                bufdata := buffer.Bytes()
                if self.option.Sync.Enabled {
                    le_msg := &SyncMessage{}
                    le_msg._type = SEGMEMT
                    le_msg.segment = msg.segment
                    le_msg.seg_buffer = bytes.NewBuffer(bufdata)
                    syncChan <- le_msg
                }
                if self.option.Record.Enabled {
                    le_msg := &RecordMessage{}
                    le_msg._target_duration = msg._target_duration
                    le_msg.segment = msg.segment
                    le_msg.seg_buffer = bytes.NewBuffer(bufdata)
                    recordChan <- le_msg
                }
                break // It's done, boy!!!
            }
        }
    }
    // Close following channels.
    close(recordChan)
    close(syncChan)
}

func (self *Synchronizer) doRequest(req *http.Request) (*http.Response, error) {
    req.Header.Set("User-Agent", self.option.User_Agent)
    resp, err := self.client.Do(req)
    if nil != err {
        log.Errorf("doRequest:> Request %s failed: %s \n", req.URL.Path, err)
    }
    return resp, err
}
