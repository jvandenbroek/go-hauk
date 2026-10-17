package config

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type AuthMethod int

const (
	AuthPassword AuthMethod = iota
	AuthHtpasswd
	AuthLDAP
)

type VelocityUnit int

const (
	KilometersPerHour VelocityUnit = iota
	MilesPerHour
	MetersPerSecond
)

type LinkStyle int

const (
	Link4Plus4Upper LinkStyle = iota
	Link4Plus4Lower
	Link4Plus4Mixed
	LinkUUIDv4
	Link16Hex
	Link16Upper
	Link16Lower
	Link16Mixed
	Link32Hex
	Link32Upper
	Link32Lower
	Link32Mixed
)

type Config struct {
	ListenAddr string
	PublicURL  string

	// redis
	RedisAddr     string
	RedisPassword string
	RedisPrefix   string

	// auth
	AuthMethod   AuthMethod
	PasswordHash string
	HtpasswdPath string

	// ldap
	LDAPUri        string
	LDAPBaseDN     string
	LDAPBindDN     string
	LDAPBindPass   string
	LDAPUserFilter string
	LDAPStartTLS   bool

	// share limits
	MaxDuration  int
	MinInterval  float64
	MaxCachedPts int
	MaxShownPts  int

	// link generation
	LinkStyle      LinkStyle
	AllowLinkReq   bool
	ReservedLinks  map[string][]string
	ReserveWL      bool

	// map display
	MapTileURI     string
	MapAttribution string
	DefaultZoom    int
	MaxZoom        int

	// velocity
	VelocityUnit      VelocityUnit
	VelocityDataPts   int
	TrailColor        string
	OfflineTimeout    int
	RequestTimeout    int
	SpeedFontSize     string

	// speed-dependent auto zoom in the web viewer
	AutoZoom            bool
	AutoZoomSteps       []ZoomStep
	AutoZoomSmoothing   float64 // EMA weight of the newest speed sample, (0, 1]
	AutoZoomMinInterval float64 // seconds between automatic zoom changes
	AutoZoomHysteresis  float64 // km/h margin around step breakpoints
	AutoZoomResume      int     // seconds after manual zoom before auto zoom resumes, 0 = never

	// rate limiting
	RateLimitAuth   int  // requests per minute for auth endpoints
	RateLimitAdopt  int  // requests per minute for adopt/join endpoints
	TrustProxy      bool // trust X-Forwarded-For headers
}

// ZoomStep maps a minimum speed in km/h to a map zoom level.
type ZoomStep struct {
	Speed float64
	Zoom  int
}

// MarshalJSON encodes a step as [speed, zoom] for the frontend.
func (z ZoomStep) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%g,%d]", z.Speed, z.Zoom)), nil
}

const defaultZoomSteps = "0:17,10:16,30:15,60:14,90:13,120:12"

// ParseZoomSteps parses "speed:zoom,..." (speed in km/h) into steps sorted
// by speed.
func ParseZoomSteps(s string) ([]ZoomStep, error) {
	var steps []ZoomStep
	for _, part := range strings.Split(s, ",") {
		speed, zoom, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			return nil, fmt.Errorf("invalid step %q", part)
		}
		sp, err := strconv.ParseFloat(strings.TrimSpace(speed), 64)
		if err != nil || sp < 0 || sp > 10000 {
			return nil, fmt.Errorf("invalid speed in step %q", part)
		}
		z, err := strconv.Atoi(strings.TrimSpace(zoom))
		if err != nil || z < 0 || z > 30 {
			return nil, fmt.Errorf("invalid zoom in step %q", part)
		}
		steps = append(steps, ZoomStep{Speed: sp, Zoom: z})
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].Speed < steps[j].Speed })
	return steps, nil
}

func envZoomSteps(key string) []ZoomStep {
	steps, err := ParseZoomSteps(envStr(key, defaultZoomSteps))
	if err != nil {
		log.Printf("%s: %v, using default", key, err)
		steps, _ = ParseZoomSteps(defaultZoomSteps)
	}
	return steps
}

var cssSizeRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(px|em|rem|pt|%)$`)

func envCSSSize(key, def string) string {
	v := envStr(key, def)
	if !cssSizeRe.MatchString(v) {
		log.Printf("%s: invalid css size %q, using %s", key, v, def)
		return def
	}
	return v
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1"
	}
	return def
}

func Load() *Config {
	authMethod := AuthPassword
	switch envStr("HAUK_AUTH_METHOD", "password") {
	case "htpasswd":
		authMethod = AuthHtpasswd
	case "ldap":
		authMethod = AuthLDAP
	}

	velUnit := KilometersPerHour
	switch envStr("HAUK_VELOCITY_UNIT", "km/h") {
	case "mph":
		velUnit = MilesPerHour
	case "m/s":
		velUnit = MetersPerSecond
	}

	cfg := &Config{
		ListenAddr:     envStr("HAUK_LISTEN_ADDR", ":8080"),
		PublicURL:      envStr("HAUK_PUBLIC_URL", "http://localhost:8080/"),
		RedisAddr:      envStr("HAUK_REDIS_ADDR", "localhost:6379"),
		RedisPassword:  envStr("HAUK_REDIS_PASSWORD", ""),
		RedisPrefix:    envStr("HAUK_REDIS_PREFIX", "hauk"),
		AuthMethod:     authMethod,
		PasswordHash:   envStr("HAUK_PASSWORD_HASH", ""),
		HtpasswdPath:   envStr("HAUK_HTPASSWD_PATH", "/etc/hauk/users.htpasswd"),
		LDAPUri:        envStr("HAUK_LDAP_URI", ""),
		LDAPBaseDN:     envStr("HAUK_LDAP_BASE_DN", ""),
		LDAPBindDN:     envStr("HAUK_LDAP_BIND_DN", ""),
		LDAPBindPass:   envStr("HAUK_LDAP_BIND_PASS", ""),
		LDAPUserFilter: envStr("HAUK_LDAP_USER_FILTER", "(uid=%s)"),
		LDAPStartTLS:   envBool("HAUK_LDAP_START_TLS", false),
		MaxDuration:    envInt("HAUK_MAX_DURATION", 86400),
		MinInterval:    envFloat("HAUK_MIN_INTERVAL", 1),
		MaxCachedPts:   envInt("HAUK_MAX_CACHED_PTS", 3),
		MaxShownPts:    envInt("HAUK_MAX_SHOWN_PTS", 100),
		LinkStyle:      LinkStyle(envInt("HAUK_LINK_STYLE", 0)),
		AllowLinkReq:   envBool("HAUK_ALLOW_LINK_REQ", true),
		ReservedLinks:  make(map[string][]string),
		ReserveWL:      envBool("HAUK_RESERVE_WHITELIST", false),
		MapTileURI:     envStr("HAUK_MAP_TILE_URI", "https://tile.openstreetmap.org/{z}/{x}/{y}.png"),
		MapAttribution: envStr("HAUK_MAP_ATTRIBUTION", `&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors`),
		DefaultZoom:    envInt("HAUK_DEFAULT_ZOOM", 14),
		MaxZoom:        envInt("HAUK_MAX_ZOOM", 19),
		VelocityUnit:   velUnit,
		VelocityDataPts: envInt("HAUK_VELOCITY_DATA_PTS", 2),
		TrailColor:     envStr("HAUK_TRAIL_COLOR", "#d80037"),
		OfflineTimeout:  envInt("HAUK_OFFLINE_TIMEOUT", 30),
		RequestTimeout:  envInt("HAUK_REQUEST_TIMEOUT", 10),
		SpeedFontSize:   envCSSSize("HAUK_SPEED_FONT_SIZE", "1.6em"),
		AutoZoom:            envBool("HAUK_AUTO_ZOOM", true),
		AutoZoomSteps:       envZoomSteps("HAUK_AUTO_ZOOM_STEPS"),
		AutoZoomSmoothing:   envFloat("HAUK_AUTO_ZOOM_SMOOTHING", 0.3),
		AutoZoomMinInterval: envFloat("HAUK_AUTO_ZOOM_MIN_INTERVAL", 10),
		AutoZoomHysteresis:  envFloat("HAUK_AUTO_ZOOM_HYSTERESIS", 5),
		AutoZoomResume:      envInt("HAUK_AUTO_ZOOM_RESUME", 300),
		RateLimitAuth:   envInt("HAUK_RATE_LIMIT_AUTH", 10),
		RateLimitAdopt:  envInt("HAUK_RATE_LIMIT_ADOPT", 10),
		TrustProxy:      envBool("HAUK_TRUST_PROXY", true),
	}
	if cfg.AutoZoomSmoothing <= 0 || cfg.AutoZoomSmoothing > 1 {
		log.Printf("HAUK_AUTO_ZOOM_SMOOTHING must be in (0, 1], using 0.3")
		cfg.AutoZoomSmoothing = 0.3
	}
	return cfg
}
