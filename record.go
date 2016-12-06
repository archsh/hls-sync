/**
	This source file contains the recording process.
 */
package main

import (
	"bytes"
	"github.com/archsh/m3u8"
	log "github.com/Sirupsen/logrus"
	"strings"
	"github.com/archsh/timefmt"
	"time"
	"regexp"
	"fmt"
	"os"
	"io"
	"path/filepath"
)

type RecordMessage struct {
	_target_duration float64
	segment *m3u8.MediaSegment
	seg_buffer *bytes.Buffer
}

type TimeStampType uint8
type IndexType uint8

const (
	TST_LOCAL TimeStampType = 1 + iota
	TST_PROGRAM
	TST_SEGMENT
)

const (
	IDXT_HOUR IndexType = 1 + iota
	IDXT_MINUTE
)

func (self *Synchronizer) recordProc(msgChan chan *RecordMessage) {
	index_by := IDXT_HOUR
	switch strings.ToLower(self.option.Record.Reindex_By) {
	case "hour":
		index_by = IDXT_HOUR
	case "minute":
		index_by = IDXT_MINUTE
	default:
		index_by = IDXT_HOUR
	}
	log.Debugln("Index By:", index_by)
	var index uint64 = 0
	var index_playlist, timeshift_playlist *m3u8.MediaPlaylist
	var e error
	last_seg_timestamp := time.Time{}
	var last_seg_duration time.Duration = 0
	_target_duration := 0
	var max_timeshift_segs uint = 0
	for msg := range msgChan {
		if nil == msg {
			continue
		}
		segtime := msg.segment.ProgramDateTime
		if _target_duration == 0 {
			if self.option.Target_Duration < 1 {
				_target_duration = int(msg._target_duration)
			}else{
				_target_duration = self.option.Target_Duration
			}
		}
		if self.option.Record.Timeshifting {
			if timeshift_playlist == nil {
				fname := filepath.Join(self.option.Record.Output, self.option.Record.Timeshift_filename)
				max_timeshift_segs = uint((time.Duration(self.option.Record.Timeshift_duration) * time.Hour) / (time.Second * time.Duration(_target_duration)))
				if ! exists(fname) {
					timeshift_playlist, e = m3u8.NewMediaPlaylist(max_timeshift_segs, max_timeshift_segs)
					if e != nil {
						log.Errorf("Create playlist '%s' for timeshifting failed:> %s \n", fname, e)
					}
				}else{
					// READ playlist.
					if f, e := os.Open(fname); e != nil {
						log.Errorf("Read timeshift playlist '%s' failed:> %s \n", fname, e)
					}else{
						if playlist, listType, err := m3u8.DecodeFrom(f, true); nil != err {
							log.Errorln("Decode previous index playlist '%s' failed:> %s\n", fname, err)
						}else{
							if listType == m3u8.MEDIA {
								timeshift_playlist = playlist.(*m3u8.MediaPlaylist)
								for _, v := range timeshift_playlist.Segments {
									if v != nil {
										last_seg_timestamp = v.ProgramDateTime
									}
								}
							}else{
								log.Warningf("Previous timeshift playlist file '%s' is not a MediaPlaylist ???\n", fname)
							}
						}
						f.Close()
					}
				}
				log.Debugf("Set Timeshift playlist winsize to : %d \n", max_timeshift_segs)
				//timeshift_playlist
				if e = timeshift_playlist.SetCapacity(max_timeshift_segs); nil != e {
					log.Errorf("SetCapacity to %d failed:> %s\n", max_timeshift_segs, e)
				}
				if e = timeshift_playlist.SetWinSize(max_timeshift_segs); nil != e {
					log.Errorf("SetWinSize to %d failed:> %s\n", max_timeshift_segs, e)
				}
			}
		}
		if nil == index_playlist {
			fname, e := self.generateFilename(self.option.Record.Output, self.option.Record.Reindex_Format, segtime, 0)
			if nil != e {
				log.Errorf("Generate index playlist '%s' failed:> %s\n", fname, e)
			}else if fname != "" && exists(fname){
				// READ playlist.
				if f, e := os.Open(fname); e != nil {
					log.Errorf("Read previous index playlist '%s' failed:> %s\n", fname, e)
				}else{
					if playlist, listType, err := m3u8.DecodeFrom(f, true); nil != err {
						log.Errorf("Decode previous index playlist '%s' failed:> %s\n", fname, err)
					}else{
						if listType == m3u8.MEDIA {
							index_playlist = playlist.(*m3u8.MediaPlaylist)
							for _, v := range index_playlist.Segments {
								if v != nil {
									last_seg_timestamp = v.ProgramDateTime
									if index_by == IDXT_MINUTE {
										index = uint64(last_seg_timestamp.Second()/_target_duration)
									}else{
										index = uint64((segtime.Minute()*60+segtime.Second())/_target_duration)
									}
								}
							}
						}else{
							log.Warningf("Previous index playlist file '%s' is not a MediaPlaylist ???\n", fname)
						}
					}
					f.Close()
				}
			}
		}
		if index_by == IDXT_MINUTE {
			index = uint64(segtime.Second()/_target_duration)
			if segtime.Year() != last_seg_timestamp.Year() ||
				segtime.Month() != last_seg_timestamp.Month() ||
				segtime.Day() != last_seg_timestamp.Day() ||
				segtime.Hour() != last_seg_timestamp.Hour() ||
				segtime.Minute() != last_seg_timestamp.Minute() {
				if self.option.Record.Reindex {
					if index_playlist != nil {
						index_playlist.Close()
						self.saveIndexPlaylist(index_playlist)
					}
					index_playlist, e = m3u8.NewMediaPlaylist(128, 128)
					if nil != e {
						log.Errorln("Create playlist failed:>", e)
						continue
					}
					index_playlist.TargetDuration = float64(_target_duration)
				}
			}
		}else{
			index = uint64((segtime.Minute()*60+segtime.Second())/_target_duration)
			if segtime.Year() != last_seg_timestamp.Year() ||
				segtime.Month() != last_seg_timestamp.Month() ||
				segtime.Day() != last_seg_timestamp.Day() ||
				segtime.Hour() != last_seg_timestamp.Hour() {
				if self.option.Record.Reindex {
					if index_playlist != nil {
						index_playlist.Close()
						self.saveIndexPlaylist(index_playlist)
					}
					index_playlist, e = m3u8.NewMediaPlaylist(2048, 2048)
					if nil != e {
						log.Errorln("Create playlist failed:>", e)
						continue
					}
					index_playlist.TargetDuration = float64(_target_duration)
				}
			}
		}
		// In case of stream paused for some time.
		if last_seg_duration > 0 && segtime.Sub(last_seg_timestamp) > time.Duration(last_seg_duration*2)*time.Second {
			if index_by == IDXT_MINUTE {
				index = uint64(segtime.Second()/_target_duration)
			}else{
				index = uint64((segtime.Minute()*60+segtime.Second())/_target_duration)
			}
		}
		log.Debugln("Recording segment:> ", msg.segment, msg.seg_buffer.Len())
		fname, e := self.generateFilename(self.option.Record.Output, self.option.Record.Segment_Rewrite, msg.segment.ProgramDateTime, index+1)
		//log.Debugf("New filename:> %s <%s> \n", fname, e)
		e = os.MkdirAll(filepath.Dir(fname), 0777)
		if e != nil {
			log.Errorf("Create directory '%s' failed:> %s \n", filepath.Dir(fname), e)
			continue
		}
		if exists(fname) {
			log.Warningf("Segment file <%s> exists! Skipped!", fname)
			continue
		}
		out, err := os.Create(fname)
		if err != nil {
			log.Errorf("Create segment file '%s' failed:> %s \n", fname, err)
			return
		}
		n, e := msg.seg_buffer.WriteTo(out)
		if nil != e {
			log.Errorf("Write to segment file '%s' failed:> %s \n", fname, err)
			out.Close()
			continue
		}else{
			log.Debugf("Write to segment file '%s' bytes:> %d \n", fname, n)
		}
		out.Close()
		last_seg_timestamp = msg.segment.ProgramDateTime
		last_seg_duration = time.Duration(msg.segment.Duration)
		log.Infof("Recorded segment:> %s | %s | %s \n", msg.segment.URI, msg.segment.ProgramDateTime, fname)
		index++
		if self.option.Record.Reindex {
			seg := m3u8.MediaSegment{
				URI: filepath.Base(fname),
				Duration: msg.segment.Duration,
				ProgramDateTime: msg.segment.ProgramDateTime,
				Title: msg.segment.URI,
				SeqId: index,
			}
			if e := index_playlist.AppendSegment(&seg); nil == e {
				self.saveIndexPlaylist(index_playlist)
			}else{
				log.Errorf("Append to index playlist failed:> %s \n", e)
			}

		}
		if self.option.Record.Timeshifting {
			if relpath, e := filepath.Rel(self.option.Record.Output, fname); nil == e {
				seg := m3u8.MediaSegment{
					URI: filepath.ToSlash(relpath),
					Duration: msg.segment.Duration,
					ProgramDateTime: msg.segment.ProgramDateTime,
					Title: msg.segment.URI,
					SeqId: index,
				}
				if timeshift_playlist.Count() >= max_timeshift_segs {
					if e:= timeshift_playlist.Remove(); nil != e {
						log.Errorln("Remove segment from timeshift playlist failed:>", e)
					}
				}
				if e := timeshift_playlist.AppendSegment(&seg); nil == e {
					self.saveTimeshiftPlaylist(timeshift_playlist)
				}else{
					log.Errorf("Append to timeshift playlist failed:> %s \n", e)
				}

			}else{
				log.Errorf("Get relative path of '%s' failed:> %s \n", fname, e)
			}
		}
	}
}

func (self *Synchronizer) saveTimeshiftPlaylist(playlist *m3u8.MediaPlaylist) {
	if nil == playlist || nil == playlist.Segments[0] {
		log.Errorln("Empty playlist !")
		return
	}
	fname := filepath.Join(self.option.Record.Output, self.option.Record.Timeshift_filename)
	log.Debugf("Updating timeshift playlist file:> %s : %d \n", fname, playlist.Count())
	e := os.MkdirAll(filepath.Dir(fname), 0777)
	if e != nil {
		log.Errorf("Create directory '%s' failed:> %s \n", filepath.Dir(fname), e)
		return
	}
	out, err := os.Create(fname)
	if err != nil {
		log.Errorf("Create timeshift file '%s' failed:>  %s \n", fname, err)
		return
	}
	defer out.Close()
	//playlist.SetWinSize(playlist.Count())
	playlist.Close()
	buf := playlist.Encode()
	n, e := io.Copy(out, buf)
	if nil != e {
		log.Errorf("Write timeshift file '%s' failed:> %s \n", fname, e)
	}else{
		log.Debugf("Write timeshift file '%s' bytes:> %d \n", fname, n)
	}
	log.Infof("Updated timeshift playlist:> %s : %d \n", fname, playlist.Count())
}

func (self *Synchronizer) saveIndexPlaylist(playlist *m3u8.MediaPlaylist) {
	if nil == playlist || nil == playlist.Segments[0] {
		log.Errorln("Empty playlist !")
		return
	}
	fname, e := self.generateFilename(self.option.Record.Output, self.option.Record.Reindex_Format, playlist.Segments[0].ProgramDateTime, 0)
	log.Debugf("Re-index into file:> %s <%s> \n", fname, e)
	e = os.MkdirAll(filepath.Dir(fname), 0777)
	if e != nil {
		log.Errorf("Create directory '%s' failed:> %s \n", filepath.Dir(fname), e)
		return
	}
	out, err := os.Create(fname)
	if err != nil {
		log.Errorf("Create index file '%s' failed:>  %s \n", fname, err)
		return
	}
	defer out.Close()
	playlist.SetWinSize(playlist.Count())
	buf := playlist.Encode()
	n, e := io.Copy(out, buf)
	if nil != e {
		log.Errorf("Write index file '%s' failed:> %s \n", fname, e)
	}else{
		log.Debugf("Write index file '%s' bytes:> %d \n", fname, n)
	}
	log.Infof("Updated index playlist:> %s \n", fname)
}

func (self *Synchronizer) generateFilename(output string, format string, tm time.Time, idx uint64) (string, error) {
	s, e := timefmt.Strftime(tm, format)
	if e != nil {
		return "", nil
	}
	re, e := regexp.Compile("(#)(:?)(\\d{0,2})")
	if e != nil {
		return "", nil
	}
	if re.MatchString(s){
		s = re.ReplaceAllString(s, "%${3}d")
		s = fmt.Sprintf(s, idx)
	}
	return filepath.Join(output, s), nil
}

func exists(path string) bool {
	s, err := os.Stat(path)
	if nil != err || s.Size() < 1 {
		return false
	}
	return true
}