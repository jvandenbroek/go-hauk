package model

import (
	"context"
	"testing"
	"time"

	"github.com/parkan/go-hauk/store"
)

func newTestSession(maxPts int, encrypted bool) (*Session, *store.Memory) {
	mem := store.NewMemory()
	return &Session{
		store:  mem,
		id:     "abc",
		maxPts: maxPts,
		data:   SessionData{Expire: time.Now().Add(time.Hour).Unix(), Encrypted: encrypted},
	}, mem
}

func TestSessionAddPointTrims(t *testing.T) {
	ctx := context.Background()
	sess, _ := newTestSession(3, false)
	for i := 0; i < 5; i++ {
		if err := sess.AddPoint(ctx, []any{float64(i), float64(i), float64(i), 0, nil, nil}); err != nil {
			t.Fatal(err)
		}
	}
	pts, err := sess.GetPoints(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 3 {
		t.Fatalf("expected 3 points, got %d", len(pts))
	}
	if pts[0][0].(float64) != 2 || pts[2][0].(float64) != 4 {
		t.Errorf("expected points 2..4, got %v", pts)
	}
}

func TestSessionGetPointsSince(t *testing.T) {
	ctx := context.Background()
	sess, _ := newTestSession(1000, false)
	for i := 1; i <= 100; i++ {
		sess.AddPoint(ctx, []any{1.0, 1.0, float64(i * 100), 0, nil, nil})
	}
	pts, _ := sess.GetPoints(ctx, 150)
	if len(pts) != 99 {
		t.Errorf("expected 99 points after since=150, got %d", len(pts))
	}
	if pts[0][2].(float64) != 200 || pts[len(pts)-1][2].(float64) != 10000 {
		t.Errorf("wrong order/bounds: first %v last %v", pts[0], pts[len(pts)-1])
	}
	pts, _ = sess.GetPoints(ctx, 9800)
	if len(pts) != 2 {
		t.Errorf("expected 2 points, got %d", len(pts))
	}
	pts, _ = sess.GetPoints(ctx, 10000)
	if len(pts) != 0 {
		t.Errorf("expected 0 points, got %d", len(pts))
	}
	pts, _ = sess.GetPoints(ctx, 0)
	if len(pts) != 100 {
		t.Errorf("expected all 100 points, got %d", len(pts))
	}
}

func TestSessionEncryptedReturnsAll(t *testing.T) {
	ctx := context.Background()
	sess, _ := newTestSession(10, true)
	for i := 0; i < 4; i++ {
		sess.AddPoint(ctx, []any{"iv", "lat", "lon", "ciphertext", nil, nil, nil})
	}
	pts, _ := sess.GetPoints(ctx, 99999)
	if len(pts) != 4 {
		t.Errorf("expected all 4 cached points for encrypted session, got %d", len(pts))
	}
}

func TestSessionDeleteRemovesPoints(t *testing.T) {
	ctx := context.Background()
	sess, mem := newTestSession(5, false)
	sess.AddPoint(ctx, []any{1.0, 1.0, 100.0, 0, nil, nil})
	if err := sess.Save(ctx); err != nil {
		t.Fatal(err)
	}
	if err := sess.Delete(ctx); err != nil {
		t.Fatal(err)
	}
	raw, _ := mem.ListRange(ctx, PrefixPoints+"abc", 0, -1)
	if len(raw) != 0 {
		t.Errorf("expected points removed, got %d", len(raw))
	}
	if ok, _ := mem.Exists(ctx, PrefixSession+"abc"); ok {
		t.Error("expected session removed")
	}
}

func TestSessionBlobHasNoPoints(t *testing.T) {
	ctx := context.Background()
	sess, mem := newTestSession(5, false)
	sess.AddPoint(ctx, []any{1.0, 1.0, 100.0, 0, nil, nil})
	sess.Save(ctx)
	var m map[string]any
	mem.Get(ctx, PrefixSession+"abc", &m)
	if _, ok := m["points"]; ok {
		t.Error("session blob must not contain points")
	}
}

func TestSessionExpired(t *testing.T) {
	sess := &Session{
		data: SessionData{
			Expire: time.Now().Add(-time.Hour).Unix(),
		},
	}

	if !sess.HasExpired() {
		t.Error("expected session to be expired")
	}

	sess.data.Expire = time.Now().Add(time.Hour).Unix()
	if sess.HasExpired() {
		t.Error("expected session to not be expired")
	}
}
