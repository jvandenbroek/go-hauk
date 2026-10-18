package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/parkan/go-hauk/config"
)

type velocityUnit struct {
	MpsMultiplier float64 `json:"mpsMultiplier"`
	Unit          string  `json:"unit"`
}

type autoZoomConfig struct {
	Enabled     bool              `json:"enabled"`
	Steps       []config.ZoomStep `json:"steps"`
	Smoothing   float64           `json:"smoothing"`
	InInterval  float64           `json:"inInterval"`
	OutInterval float64           `json:"outInterval"`
	Jump        int               `json:"jump"`
	MaxAccuracy float64           `json:"maxAccuracy"`
	Hysteresis  float64           `json:"hysteresis"`
	Resume      int               `json:"resume"`
}

func (s *Server) handleDynamic(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")

	var velUnit velocityUnit
	switch s.cfg.VelocityUnit {
	case config.MilesPerHour:
		velUnit = velocityUnit{MpsMultiplier: 3.6 * 0.6213712, Unit: "mph"}
	case config.MetersPerSecond:
		velUnit = velocityUnit{MpsMultiplier: 1, Unit: "m/s"}
	default:
		velUnit = velocityUnit{MpsMultiplier: 3.6, Unit: "km/h"}
	}

	tileURI, _ := json.Marshal(s.cfg.MapTileURI)
	attribution, _ := json.Marshal(s.cfg.MapAttribution)
	defaultZoom, _ := json.Marshal(s.cfg.DefaultZoom)
	maxZoom, _ := json.Marshal(s.cfg.MaxZoom)
	maxPoints, _ := json.Marshal(s.cfg.MaxShownPts)
	velDelta, _ := json.Marshal(s.cfg.VelocityDataPts)
	trailColor, _ := json.Marshal(s.cfg.TrailColor)
	velUnitJSON, _ := json.Marshal(velUnit)
	offlineTimeout, _ := json.Marshal(s.cfg.OfflineTimeout)
	requestTimeout, _ := json.Marshal(s.cfg.RequestTimeout)
	speedFontSize, _ := json.Marshal(s.cfg.SpeedFontSize)
	lookahead, _ := json.Marshal(s.cfg.Lookahead)
	autoZoom, _ := json.Marshal(autoZoomConfig{
		Enabled:     s.cfg.AutoZoom && len(s.cfg.AutoZoomSteps) > 0,
		Steps:       s.cfg.AutoZoomSteps,
		Smoothing:   s.cfg.AutoZoomSmoothing,
		InInterval:  s.cfg.AutoZoomMinInterval,
		OutInterval: s.cfg.AutoZoomOutInterval,
		Jump:        s.cfg.AutoZoomJump,
		MaxAccuracy: s.cfg.AutoZoomMaxAccuracy,
		Hysteresis:  s.cfg.AutoZoomHysteresis,
		Resume:      s.cfg.AutoZoomResume,
	})

	fmt.Fprintf(w, "var TILE_URI = %s;\n", tileURI)
	fmt.Fprintf(w, "var ATTRIBUTION = %s;\n", attribution)
	fmt.Fprintf(w, "var DEFAULT_ZOOM = %s;\n", defaultZoom)
	fmt.Fprintf(w, "var MAX_ZOOM = %s;\n", maxZoom)
	fmt.Fprintf(w, "var MAX_POINTS = %s;\n", maxPoints)
	fmt.Fprintf(w, "var VELOCITY_DELTA_TIME = %s;\n", velDelta)
	fmt.Fprintf(w, "var TRAIL_COLOR = %s;\n", trailColor)
	fmt.Fprintf(w, "var VELOCITY_UNIT = %s;\n", velUnitJSON)
	fmt.Fprintf(w, "var OFFLINE_TIMEOUT = %s;\n", offlineTimeout)
	fmt.Fprintf(w, "var REQUEST_TIMEOUT = %s;\n", requestTimeout)
	fmt.Fprintf(w, "var SPEED_FONT_SIZE = %s;\n", speedFontSize)
	fmt.Fprintf(w, "var AUTO_ZOOM = %s;\n", autoZoom)
	fmt.Fprintf(w, "var LOOKAHEAD = %s;\n", lookahead)
}
