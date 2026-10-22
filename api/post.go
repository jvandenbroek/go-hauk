package api

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/parkan/go-hauk/model"
)

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	sid := r.FormValue("sid")
	lat := r.FormValue("lat")
	lon := r.FormValue("lon")
	ts := r.FormValue("time")

	if sid == "" || lat == "" || lon == "" || ts == "" {
		fmt.Fprintln(w, "Missing data!")
		return
	}

	session, err := model.LoadSession(ctx, s.store, sid, s.cfg.MaxCachedPts)
	if err != nil || session.HasExpired() {
		fmt.Fprintln(w, "Session expired!")
		return
	}

	var point []any

	if !session.Encrypted() {
		latF, err1 := strconv.ParseFloat(lat, 64)
		lonF, err2 := strconv.ParseFloat(lon, 64)
		timeF, err3 := strconv.ParseFloat(ts, 64)

		// ParseFloat accepts NaN/Inf, which pass the range check but break
		// JSON encoding of the session
		if err1 != nil || err2 != nil || err3 != nil || !finite(latF, lonF, timeF) ||
			latF < -90 || latF > 90 || lonF < -180 || lonF > 180 {
			fmt.Fprintln(w, "Invalid location!")
			return
		}

		var speed, acc *float64
		if spd := r.FormValue("spd"); spd != "" {
			if v, err := strconv.ParseFloat(spd, 64); err == nil && finite(v) {
				speed = &v
			}
		}
		if a := r.FormValue("acc"); a != "" {
			if v, err := strconv.ParseFloat(a, 64); err == nil && finite(v) {
				acc = &v
			}
		}

		prv := 0
		if r.FormValue("prv") == "1" {
			prv = 1
		}

		point = []any{latF, lonF, timeF, prv, acc, speed}
	} else {
		iv := r.FormValue("iv")
		if iv == "" {
			fmt.Fprintln(w, "Missing data!")
			return
		}
		// opaque ciphertext can't be validated, but bound its size
		for _, k := range []string{"iv", "lat", "lon", "time", "spd", "acc", "prv"} {
			if len(r.FormValue(k)) > maxEncryptedFieldLen {
				fmt.Fprintln(w, "Invalid location!")
				return
			}
		}

		var speed, acc, prv any
		if spd := r.FormValue("spd"); spd != "" {
			speed = spd
		}
		if a := r.FormValue("acc"); a != "" {
			acc = a
		}
		if p := r.FormValue("prv"); p != "" {
			prv = p
		}

		point = []any{iv, lat, lon, ts, prv, acc, speed}
	}

	if err := session.AddPoint(ctx, point); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "%s?%%s\n", s.cfg.PublicURL)
	fmt.Fprintln(w, strings.Join(session.Targets(), ","))
}

const maxEncryptedFieldLen = 256

func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
