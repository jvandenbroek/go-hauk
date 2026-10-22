package model

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/parkan/go-hauk/store"
)

const (
	PrefixSession = "session-"
	// PrefixPoints keys a Redis list (one JSON point per element) per session
	PrefixPoints = "points-"
	SessionIDSize = 32
)

type SessionData struct {
	Expire    int64    `json:"expire"`
	Interval  float64  `json:"interval"`
	Targets   []string `json:"targets"`
	Encrypted bool     `json:"encrypted"`
	Salt      string   `json:"salt,omitempty"`
}

type Session struct {
	store store.Store
	id    string
	data  SessionData
	maxPts int
}

func NewSession(s store.Store, maxPts int) (*Session, error) {
	id, err := generateSessionID()
	if err != nil {
		return nil, err
	}
	return &Session{
		store:  s,
		id:     id,
		maxPts: maxPts,
		data: SessionData{
			Targets: []string{},
		},
	}, nil
}

func LoadSession(ctx context.Context, s store.Store, id string, maxPts int) (*Session, error) {
	sess := &Session{store: s, id: id, maxPts: maxPts}
	err := s.Get(ctx, PrefixSession+id, &sess.data)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *Session) ID() string               { return s.id }
func (s *Session) Expire() time.Time        { return time.Unix(s.data.Expire, 0) }
func (s *Session) Interval() float64        { return s.data.Interval }
func (s *Session) Targets() []string        { return s.data.Targets }
func (s *Session) Encrypted() bool          { return s.data.Encrypted }
func (s *Session) Salt() string             { return s.data.Salt }
func (s *Session) HasExpired() bool         { return time.Now().Unix() >= s.data.Expire }

func (s *Session) SetExpire(t time.Time)    { s.data.Expire = t.Unix() }
func (s *Session) SetInterval(i float64)    { s.data.Interval = i }
func (s *Session) SetEncrypted(e bool, salt string) {
	s.data.Encrypted = e
	s.data.Salt = salt
}

func (s *Session) AddTarget(shareID string) {
	s.data.Targets = append(s.data.Targets, shareID)
}

func (s *Session) RemoveTarget(shareID string) {
	for i, t := range s.data.Targets {
		if t == shareID {
			s.data.Targets = append(s.data.Targets[:i], s.data.Targets[i+1:]...)
			return
		}
	}
}

// AddPoint appends only the new point to the session's points list and trims
// it to maxPts. The session metadata blob is not touched.
func (s *Session) AddPoint(ctx context.Context, p []any) error {
	return s.store.ListPush(ctx, PrefixPoints+s.id, p, s.maxPts, s.Expire())
}

// pointChunk is how many points are read per round trip when scanning from
// the newest point backwards.
const pointChunk = 32

func (s *Session) GetPoints(ctx context.Context, since float64) ([][]any, error) {
	key := PrefixPoints + s.id
	// encrypted sessions have opaque timestamps - can't filter server-side
	if since <= 0 || s.data.Encrypted {
		raw, err := s.store.ListRange(ctx, key, 0, -1)
		if err != nil {
			return nil, err
		}
		return decodePoints(raw), nil
	}
	// points are assumed to be appended in time order: read from the tail in
	// chunks and stop at the first point not newer than since. A late point
	// behind newer ones is not returned here; the frontend drops such points
	// too (time must exceed the last drawn point), so nothing visible is lost.
	var newest [][]any // collected newest-first
	for off := 0; ; off += pointChunk {
		raw, err := s.store.ListRange(ctx, key, -off-pointChunk, -off-1)
		if err != nil {
			return nil, err
		}
		pts := decodePoints(raw)
		done := len(raw) < pointChunk
		for i := len(pts) - 1; i >= 0; i-- {
			p := pts[i]
			if len(p) > 2 {
				if t, ok := p[2].(float64); ok && t > since {
					newest = append(newest, p)
					continue
				}
			}
			done = true
			break
		}
		if done {
			break
		}
	}
	for i, j := 0, len(newest)-1; i < j; i, j = i+1, j-1 {
		newest[i], newest[j] = newest[j], newest[i]
	}
	return newest, nil
}

func decodePoints(raw []json.RawMessage) [][]any {
	pts := make([][]any, 0, len(raw))
	for _, r := range raw {
		var p []any
		if json.Unmarshal(r, &p) == nil {
			pts = append(pts, p)
		}
	}
	return pts
}

func (s *Session) Save(ctx context.Context) error {
	return s.store.Set(ctx, PrefixSession+s.id, s.data, s.Expire())
}

func (s *Session) Delete(ctx context.Context) error {
	if err := s.store.Delete(ctx, PrefixPoints+s.id); err != nil {
		return err
	}
	return s.store.Delete(ctx, PrefixSession+s.id)
}

func generateSessionID() (string, error) {
	b := make([]byte, SessionIDSize)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
