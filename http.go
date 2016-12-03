/***
	This source file contains http service processing.
	HTTP service provides a time-shifting playlist access and generating.
	GET /?playlist={start-timestamp}_{end-timestamp}.m3u8           eg: /?playlist=1479998100_1480004640.m3u8
	GET /?start={start-timestamp}&duration={duration-in-seconds}    eg: /?start=1479998100&duration=6540
	GET /?start={start-timestamp}&end={end-timestamp}               eg: /?start=1479998100&end=1480004640
 */
package main

import (
	"net/http"
	"time"
	log "github.com/Sirupsen/logrus"
	"strconv"
	"fmt"
	"regexp"
	"github.com/archsh/m3u8"
	"net"
	"strings"
	"os"
	"path/filepath"
	"bytes"
	"github.com/golang/groupcache/lru"
)

type CacheItem struct {
	_timestamp time.Time
	_content   []byte
}


func (self *Synchronizer) HttpServe() {
	ls := strings.Split(self.option.Http.Listen, "://")
	if len(ls) != 2 {
		log.Errorf("Invalid listen option:> '%s', should use like 'tcp://0.0.0.0:8080' or 'unix:///var/run/test.sock'.", self.option.Http.Listen)
		return
	}
	if ls[0] == "unix" {
		if e := os.Remove(ls[1]); nil != e {
			log.Errorf("Remove previous sock file '%s' failed:> %s \n", ls[1], e)
		}else{
			log.Debugf("Removed previous sock file '%s'.\n", ls[1])
		}
	}
	ln, err := net.Listen(ls[0], ls[1])
	if nil != err {
		log.Errorln("Listen to socket failed:> ", err)
	}
	self.httpCache = lru.New(self.option.Http.Cache_Num)
	e := http.Serve(ln, self)
	log.Errorln("HTTP serve failed:> ", e)
}


func (self *Synchronizer) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	_bad_request := func (msg string){
		log.Debugln("Bad Request:> ", msg)
		response.WriteHeader(400)
		response.Header().Set("Content-Type", "text/plain")
		response.Write([]byte(msg))
	}
	if request.Method != "GET" {
		_bad_request("Invalid Request Method!\n")
		return
	}
	playlist := request.URL.Query().Get("playlist")
	start := request.URL.Query().Get("start")
	duration := request.URL.Query().Get("duration")
	end := request.URL.Query().Get("end")
	var _start_time, _end_time time.Time
	if playlist != "" {
		re := regexp.MustCompile("([0-9]+)[-_]([0-9]+).m3u8")
		if !re.MatchString(playlist) {
			_bad_request(fmt.Sprintf("Invalid playlist name format : %s \n",playlist))
			return
		}
		ss := re.FindStringSubmatch(playlist)
		start = ss[1]
		end = ss[2]
	}
	if start != "" {
		_start_sec, e := strconv.ParseInt(start, 10, 64)
		if e != nil {
			_bad_request(fmt.Sprintf("Invalid 'start' parameter: '%s' \n", start))
			return
		}
		_start_time = time.Unix(_start_sec, 0)
		if end != "" {
			_end_sec, e := strconv.ParseInt(end, 10, 64)
			if e != nil {
				_bad_request(fmt.Sprintf("Invalid 'end' parameter: '%s' \n", end))
				return
			}
			_end_time = time.Unix(_end_sec, 0)
		}else if duration != "" {
			_duration_sec, e := strconv.ParseInt(duration, 10, 64)
			if e != nil {
				_bad_request(fmt.Sprintf("Invalid 'duration' parameter: '%s' \n", duration))
				return
			}
			_end_time = _start_time.Add(time.Duration(_duration_sec)*time.Second)
		}else {
			_bad_request("Missing Query Parameter 'duration' or 'end'!\n")
			return
		}
	}else{
		_bad_request("Unknown Query Parameter!\n")
		return
	}
	// Need: Start Timestamp, End Timestamp
	if _start_time.After(_end_time) || _start_time.Equal(_end_time) {
		_bad_request("Start timestamp can not be after end timestamp or as the same as end timestamp.!!!\n")
		return
	}else if time.Now().Sub(_start_time) > time.Duration(self.option.Http.Days*24)*time.Hour {
		_bad_request(fmt.Sprintf("Can not provide shifting before %d days!", self.option.Http.Days))
		return
	}else if _end_time.Sub(_start_time) > time.Duration(self.option.Http.Max)*time.Hour {
		_bad_request(fmt.Sprintf("Can not provide playlist larger than %d hours!", self.option.Http.Max))
		return
	}
	log.Infof("Request Playlist %s -> %s \n", _start_time, _end_time)
	c_key := fmt.Sprintf("%d-%d", _start_time.Unix(), _end_time.Unix())
	if v, ok := self.httpCache.Get(c_key); ok {
		log.Debugln("Cached: ", c_key)
		if item, yes := v.(CacheItem); yes {
			if item._timestamp.Add(time.Duration(self.option.Http.Cache_Valid)*time.Second).After(time.Now()) {
				response.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
				response.Header().Set("Content-Length", fmt.Sprintf("%d",len(item._content)))
				response.Write(item._content)
				return
			}
		}
	}
	if mpl, e := self.buildPlaylist(_start_time, _end_time); e != nil {
		log.Errorf("Build playlist failed:> %s \n", e)
		response.WriteHeader(500)
		response.Header().Set("Content-Type", "text/plain")
		response.Write([]byte(fmt.Sprintf("Build playlist failed:> %s", e)))
		return
	}else{
		buf := &bytes.Buffer{}
		mpl.Encode().WriteTo(buf)
		pbytes := buf.Bytes()
		response.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		response.Header().Set("Content-Length", fmt.Sprintf("%d",len(pbytes)))
		response.Write(pbytes)
		self.httpCache.Add(c_key, CacheItem{_timestamp:time.Now(), _content:pbytes})
	}
}

func (self *Synchronizer) buildPlaylist(start time.Time, end time.Time) (*m3u8.MediaPlaylist, error) {
	var duration time.Duration
	if strings.ToLower(self.option.Record.Reindex_By) == "minute" {
		duration = time.Minute
	} else {
		duration = time.Hour
	}
	mpl, e := m3u8.NewMediaPlaylist(2048,2048)
	if  nil != e {
		log.Errorf("Create MediaPlaylist failed:> %s\n", e)
		return nil, e
	}
	for t := start.Truncate(duration); t.Before(end); t = t.Add(duration) {
		log.Debugln("T:>", t)
		if index_filename, e := self.generateFilename(self.option.Record.Output, self.option.Record.Reindex_Format, t, 0); nil != e {
			log.Errorf("Generate filename failed:> %s \n", e)
			continue
		}else{
			rl_idx, _ := self.generateFilename(self.option.Http.Segment_Prefix, self.option.Record.Reindex_Format, t, 0)
			rl_path := filepath.Dir(rl_idx)
			fp, e := os.Open(index_filename)
			if nil != e {
				log.Errorf("Open index file '%s' failed:> %s \n", index_filename, e)
				continue
			}
			l, t, e := m3u8.DecodeFrom(fp, true)
			fp.Close()
			if nil != e || t != m3u8.MEDIA {
				log.Errorf("Decode index file '%s' failed:> %s \n", index_filename, e)
				continue
			}
			index_mpl := l.(*m3u8.MediaPlaylist)
			for _, seg := range index_mpl.Segments {
				if nil == seg {
					continue
				}
				if seg.ProgramDateTime.Before(start) || seg.ProgramDateTime.After(end){
					log.Debugln("Ignored segment: ", seg.URI, seg.ProgramDateTime, start, end, seg.ProgramDateTime.Before(start) , seg.ProgramDateTime.After(end))
					continue
				}
				seg.URI = filepath.ToSlash(filepath.Join(rl_path, seg.URI))
				mpl.AppendSegment(seg)
				mpl.TargetDuration = seg.Duration
			}
		}
	}
	mpl.Close()
	mpl.SetWinSize(mpl.Count())
	return mpl, nil
}
