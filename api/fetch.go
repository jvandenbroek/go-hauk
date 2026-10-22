package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/parkan/go-hauk/model"
)

type soloResponse struct {
	Type       int       `json:"type"`
	Expire     int64     `json:"expire"`
	ServerTime float64   `json:"serverTime"`
	Interval   float64   `json:"interval"`
	Points     [][]any   `json:"points"`
	Encrypted  bool      `json:"encrypted"`
	Salt       string    `json:"salt"`
	Viewers    int       `json:"viewers"`
}

type groupResponse struct {
	Type       int                 `json:"type"`
	Expire     int64               `json:"expire"`
	ServerTime float64             `json:"serverTime"`
	Interval   float64             `json:"interval"`
	Points     map[string][][]any  `json:"points"`
	Viewers    int                 `json:"viewers"`
}

func (s *Server) handleFetch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.URL.Query().Get("id")
	if id == "" {
		fmt.Fprintln(w, "Invalid session!")
		return
	}

	sinceStr := r.URL.Query().Get("since")
	since := float64(0)
	if sinceStr != "" {
		since, _ = strconv.ParseFloat(sinceStr, 64)
	}

	shareType, err := model.LoadShareType(ctx, s.store, id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Invalid session!")
		return
	}

	viewers := s.viewers.touch(id, r.URL.Query().Get("v"))

	w.Header().Set("Content-Type", "text/json")
	now := float64(time.Now().UnixNano()) / 1e9

	switch shareType {
	case model.ShareTypeAlone:
		share, err := model.LoadSoloShare(ctx, s.store, id, s.cfg.PublicURL)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "Invalid session!")
			return
		}

		session, err := model.LoadSession(ctx, s.store, share.Host(), s.cfg.MaxCachedPts)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "Invalid session!")
			return
		}

		pts, err := session.GetPoints(ctx, since)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := soloResponse{
			Type:       share.Type(),
			Expire:     share.Expire().Unix(),
			ServerTime: now,
			Interval:   session.Interval(),
			Points:     pts,
			Encrypted:  session.Encrypted(),
			Salt:       session.Salt(),
			Viewers:    viewers,
		}
		if resp.Points == nil {
			resp.Points = [][]any{}
		}
		json.NewEncoder(w).Encode(resp)

	case model.ShareTypeGroup:
		share, err := model.LoadGroupShare(ctx, s.store, id, s.cfg.PublicURL)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "Invalid session!")
			return
		}

		points, _ := share.GetAllPoints(ctx, since, s.cfg.MaxCachedPts)
		if points == nil {
			points = make(map[string][][]any)
		}

		resp := groupResponse{
			Type:       share.Type(),
			Expire:     share.Expire().Unix(),
			ServerTime: now,
			Interval:   share.GetAutoInterval(ctx, s.cfg.MaxCachedPts),
			Points:     points,
			Viewers:    viewers,
		}
		json.NewEncoder(w).Encode(resp)
	}
}
