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
	cfg := Load()
	if !cfg.AutoZoom || len(cfg.AutoZoomSteps) != 6 || cfg.AutoZoomSmoothing != 0.3 || cfg.SpeedFontSize != "1.6em" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}
