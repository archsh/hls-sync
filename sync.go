/**
	This source file contains the sync process.
 */
package main

import (
    "bytes"
    "github.com/archsh/go.m3u8"
    log "github.com/Sirupsen/logrus"
    "os"
    "path/filepath"
    "io"
    "github.com/golang/groupcache/lru"
    "io/ioutil"
)

type SyncType uint8

const (
    PLAYLIST SyncType = 1 + iota
    SEGMEMT
)

type SyncMessage struct {
    _type      SyncType
    playlist   *m3u8.MediaPlaylist
    segment    *m3u8.MediaSegment
    seg_buffer *bytes.Buffer
}

func (self *Synchronizer) syncProc(msgChan chan *SyncMessage) {
    cache := lru.New(self.option.Max_Segments)

    if self.option.Sync.Remove_Old {
        cache.OnEvicted = func(k lru.Key, v interface{}) {
            fname := v.(string)
            err := os.Remove(fname)
            if err != nil {
                log.Errorf("Delete file '%s' failed:> %s \n", fname, err)
            } else {
                log.Infof("Removed synced segment:> %s \n", fname)
            }
        }
    }
    if self.option.Sync.Clean_Folder && self.option.Sync.Output != "" && self.option.Sync.Output != "." && self.option.Sync.Output != "/" {
        // Clean target folder first.
        if filenames, e := ioutil.ReadDir(self.option.Sync.Output); nil != e {
            log.Errorf("Failed to open folder '%s' :> %s \n", self.option.Sync.Output, e)
        } else {
            for _, finfo := range filenames {
                if finfo.IsDir() {
                    continue
                }
                fname := filepath.Join(self.option.Sync.Output, finfo.Name())
                e := os.Remove(fname)
                if e != nil {
                    log.Errorf("Clear file '%s' failed:> %s\n", fname, e)
                } else {
                    log.Debugf("Cleared file '%s' ! \n", fname)
                }
            }
        }
    }

    for msg := range msgChan {
        if nil == msg {
            continue
        }
        switch msg._type {
        case PLAYLIST:
            log.Debugln("Syncing playlist:> ", msg.playlist.SeqNo, len(msg.playlist.Segments))
            filename := filepath.Join(self.option.Sync.Output, self.option.Sync.Index_Name)
            out, err := os.Create(filename)
            if err != nil {
                log.Errorf("Create playlist file '%s' failed:> %s \n", filename, err)
                continue
            }
            buf := msg.playlist.Encode()
            _, e := io.Copy(out, buf)
            if nil != e {
                log.Errorf("Write playlist '%s' failed:> %s \n", filename, e)
            }
            out.Close()
            log.Infof("Synced playlist:> %s \n", filename)
        case SEGMEMT:
            log.Debugln("Syncing segment:> ", msg.segment.URI, msg.seg_buffer.Len())
            filename := filepath.Join(self.option.Sync.Output, msg.segment.URI)
            out, err := os.Create(filename)
            if err != nil {
                log.Errorf("Create file '%s' failed:> %s \n", filename, err)
                continue
            }
            n, e := msg.seg_buffer.WriteTo(out)
            if e != nil {
                log.Errorf("Write segment file '%s' failed:> %s \n", filename, e)
            } else {
                log.Debugf("Write segment file '%s' bytes:> %d \n", filename, n)
            }
            cache.Add(msg.segment.URI, filename)
            out.Close()
            log.Infof("Synced segment:> %s | %f | %s | %s \n", msg.segment.URI, msg.segment.Duration, msg.segment.ProgramDateTime, filename)
        }
    }
}
