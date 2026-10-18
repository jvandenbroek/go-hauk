package config

import (
	"encoding/json"
	"testing"
)

func TestParseZoomSteps(t *testing.T) {
	steps, err := ParseZoomSteps("30:15, 0:17,10:16")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(steps)
	if string(b) != "[[0,17],[10,16],[30,15]]" {
		t.Errorf("unexpected steps: %s", b)
	}

	for _, bad := range []string{"", "10", "a:5", "10:x", "-1:5", "10:31"} {
		if _, err := ParseZoomSteps(bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestAutoZoomDefaults(t *testing.T) {
	t.Setenv("HAUK_AUTO_ZOOM_STEPS", "garbage")
	t.Setenv("HAUK_AUTO_ZOOM_SMOOTHING", "2")
	t.Setenv("HAUK_SPEED_FONT_SIZE", "1em;color:red")
	t.Setenv("HAUK_LOOKAHEAD", "0.9")
	cfg := Load()
	if cfg.AutoZoomOutInterval != 3 || cfg.AutoZoomJump != 2 || cfg.AutoZoomMaxAccuracy != 30 || cfg.Lookahead != 0 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if !cfg.AutoZoom || len(cfg.AutoZoomSteps) != 6 || cfg.AutoZoomSmoothing != 0.3 || cfg.SpeedFontSize != "1.6em" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestAutoZoomOutInterval(t *testing.T) {
	// the live setup only sets the (zoom-in) min interval
	t.Setenv("HAUK_AUTO_ZOOM_MIN_INTERVAL", "15")
	if cfg := Load(); cfg.AutoZoomMinInterval != 15 || cfg.AutoZoomOutInterval != 5 {
		t.Errorf("unexpected intervals: in %v out %v", cfg.AutoZoomMinInterval, cfg.AutoZoomOutInterval)
	}
	t.Setenv("HAUK_AUTO_ZOOM_OUT_INTERVAL", "2")
	if cfg := Load(); cfg.AutoZoomOutInterval != 2 {
		t.Errorf("unexpected out interval: %v", cfg.AutoZoomOutInterval)
	}
}
