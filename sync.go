/**
This source file contains the sync process.
*/
package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/archsh/go.m3u8"
	"github.com/golang/groupcache/lru"
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

func (synchron *Synchronizer) syncProc(msgChan chan *SyncMessage) {
	cache := lru.New(synchron.option.MaxSegments)

	if synchron.option.Sync.RemoveOld {
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
	if synchron.option.Sync.CleanFolder && synchron.option.Sync.Output != "" && synchron.option.Sync.Output != "." && synchron.option.Sync.Output != "/" {
		// Clean target folder first.
		if filenames, e := ioutil.ReadDir(synchron.option.Sync.Output); nil != e {
			log.Errorf("Failed to open folder '%s' :> %s \n", synchron.option.Sync.Output, e)
		} else {
			for _, finfo := range filenames {
				if finfo.IsDir() {
					continue
				}
				fname := filepath.Join(synchron.option.Sync.Output, finfo.Name())
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
			log.Debugln("Syncing playlist:> ", msg.playlist.SeqNo, len(msg.playlist.Segments), msg.playlist.Count())
			filename := filepath.Join(synchron.option.Sync.Output, synchron.option.Sync.IndexName)
			out, err := os.Create(filename)
			if err != nil {
				log.Errorf("Create playlist file '%s' failed:> %s \n", filename, err)
				continue
			}
			_ = msg.playlist.SetWinSize(msg.playlist.Count())
			buf := msg.playlist.Encode()
			log.Debugln("Writing playlist in bytes:", buf.Len(), msg.playlist.WinSize())
			if n, e := io.Copy(out, buf); nil != e {
				log.Errorf("Write playlist '%s' failed:> %s \n", filename, e)
			} else {
				log.Debugln("Wrote playlist in bytes:", n)
			}
			if e := out.Sync(); nil != e {
				log.Errorf("Sync playlist '%s' failed:> %s \n", filename, e)
			}
			if e := out.Close(); nil != e {
				log.Errorf("Close playlist '%s' failed:> %s \n", filename, e)
			}
			log.Infof("Synced playlist:> %s \n", filename)
		case SEGMEMT:
			log.Debugln("Syncing segment:> ", msg.segment.URI, msg.seg_buffer.Len())
			filename := filepath.Join(synchron.option.Sync.Output, msg.segment.URI)
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
