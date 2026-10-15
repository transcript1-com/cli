package fakeapi

import (
	"fmt"
	"slices"
	"time"

	"github.com/transcript1/cli/internal/api"
)

type job struct {
	id          string
	status      string
	platform    string
	url         string
	videoID     string
	title       string
	filename    string
	size        int64
	source      string
	language    string
	diarize     bool
	idemKey     string
	params      string
	token       string
	chunks      []api.Chunk
	received    map[int]bool
	data        []byte
	polls       int
	charged     int
	errCode     string
	createdAt   time.Time
	completedAt time.Time
}

func (s *Server) addJob(j *job) {
	s.nextID++
	j.id = fmt.Sprintf("tr_%d", s.nextID)
	j.createdAt = time.Now().UTC()
	s.jobs[j.id] = j
	s.order = append(s.order, j)
}

func (s *Server) findByKey(key string) *job {
	if key == "" {
		return nil
	}

	for _, j := range s.order {
		if j.idemKey == key {
			return j
		}
	}

	return nil
}

func isTerminal(status string) bool {
	return slices.Contains([]string{api.StatusCompleted, api.StatusFailed, api.StatusCanceled}, status)
}

func (s *Server) countActive() int {
	n := 0
	for _, j := range s.order {
		if j.status == api.StatusAwaitingUpload {
			continue
		}

		if isTerminal(j.status) {
			continue
		}

		n++
	}

	return n
}

func (s *Server) isOverActiveLimit() bool {
	if s.ActiveLimit <= 0 {
		return false
	}

	return s.countActive() >= s.ActiveLimit
}

func (s *Server) recordPeak() {
	s.peakActive = max(s.peakActive, s.countActive())
}

func (s *Server) isMinutesShort() bool {
	if s.MinutesLeft < 0 {
		return false
	}

	return s.MinutesLeft < s.MinutesPerJob
}

func (s *Server) charge(j *job) {
	if s.MinutesLeft >= 0 {
		s.MinutesLeft -= s.MinutesPerJob
	}

	j.charged = s.MinutesPerJob
}

func (s *Server) refund(j *job) {
	if s.MinutesLeft >= 0 {
		s.MinutesLeft += j.charged
	}

	j.charged = 0
}

func (s *Server) poll(j *job) {
	if isTerminal(j.status) {
		return
	}

	if j.status == api.StatusAwaitingUpload {
		return
	}

	j.polls++
	if j.polls < s.Polls {
		j.status = api.StatusTranscribing
		return
	}

	j.completedAt = time.Now().UTC()
	if s.FailCode != "" {
		j.status = api.StatusFailed
		j.errCode = s.FailCode
		s.refund(j)

		return
	}

	j.status = api.StatusCompleted
}

func (s *Server) issueToken(j *job) {
	s.nextToken++
	j.token = fmt.Sprintf("ut_%s_%d", j.id, s.nextToken)
}

func planChunks(size, chunkSize int64) []api.Chunk {
	var chunks []api.Chunk
	for offset := int64(0); offset < size; offset += chunkSize {
		chunk := api.Chunk{
			Index:  len(chunks),
			Offset: offset,
			Size:   min(chunkSize, size-offset),
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

func (j *job) listReceived() []int {
	var list []int
	for index := range j.received {
		list = append(list, index)
	}

	slices.Sort(list)

	return list
}

func (j *job) listMissing() []int {
	list := []int{}
	for _, c := range j.chunks {
		if !j.received[c.Index] {
			list = append(list, c.Index)
		}
	}

	return list
}
