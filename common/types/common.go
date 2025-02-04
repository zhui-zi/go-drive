package types

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type M map[string]interface{}
type SM map[string]string

type SV string

type TaskCtx interface {
	context.Context
	Progress(loaded int64, abs bool)
	Total(total int64, abs bool)
}

type IDisposable interface {
	Dispose() error
}

type IStatistics interface {
	// Status returns the name, status of this component
	Status() (string, SM, error)
}

// ISysConfig provides some configuration for the client
type ISysConfig interface {
	// SysConfig returns the config name, config map
	SysConfig() (string, M, error)
}

type FormItemOption struct {
	Name     string `json:"name" i18n:""`
	Title    string `json:"title" i18n:""`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type FormItem struct {
	Label        string           `json:"label" i18n:""`
	Type         string           `json:"type"`
	Field        string           `json:"field"`
	Required     bool             `json:"required"`
	Description  string           `json:"description" i18n:""`
	Disabled     bool             `json:"disabled"`
	Options      []FormItemOption `json:"options"`
	DefaultValue string           `json:"defaultValue"`
	// Secret is the replacement text when sending the value to client.
	// The raw value will be sent if Secret is empty.
	// FormItem with type 'password' will always be replaced.
	// This value cannot be used with i18n.
	Secret string `json:"-"`
}

func (s SV) Int(defVal int) int {
	v, e := strconv.ParseInt(string(s), 10, 32)
	if e != nil {
		return defVal
	}
	return int(v)
}

func (s SV) Uint(defVal uint) uint {
	v, e := strconv.ParseUint(string(s), 10, 32)
	if e != nil {
		return defVal
	}
	return uint(v)
}

func (s SV) Int64(defVal int64) int64 {
	v, e := strconv.ParseInt(string(s), 10, 64)
	if e != nil {
		return defVal
	}
	return v
}

func (s SV) Uint64(defVal uint64) uint64 {
	v, e := strconv.ParseInt(string(s), 10, 64)
	if e != nil {
		return defVal
	}
	return uint64(v)
}

func (s SV) Float64(defVal float64) float64 {
	v, e := strconv.ParseFloat(string(s), 64)
	if e != nil {
		return defVal
	}
	return v
}

func (s SV) Duration(defVal time.Duration) time.Duration {
	dur, e := time.ParseDuration(string(s))
	if e != nil {
		dur = defVal
	}
	return dur
}

func (s SV) UnixTime(defVal *time.Time) time.Time {
	if defVal == nil {
		defVal = &time.Time{}
	}
	t := s.Int64(-1)
	if t == -1 {
		return *defVal
	}
	return time.Unix(t, 0)
}

var sizeRegexp = regexp.MustCompile("^([0-9]+)([bkmgtBKMGT]?)$")
var sizeMultiplier = map[string]float64{
	"":  1,
	"b": 1,
	"k": 1024,
	"m": 1024 * 1024,
	"g": 1024 * 1024 * 1024,
	"t": 1024 * 1024 * 1024 * 1024,
}

func (s SV) DataSize(defVal int64) int64 {
	m := sizeRegexp.FindStringSubmatch(string(s))
	if m == nil {
		return defVal
	}
	size := SV(m[1]).Float64(0)
	unit := strings.ToLower(m[2])
	return int64(sizeMultiplier[unit] * size)
}

func (s SV) Bool() bool {
	v := strings.ToLower(strings.TrimSpace(string(s)))
	return s != "" && v != "false"
}

func (c SM) GetInt(key string, defVal int) int {
	return SV(c[key]).Int(defVal)
}

func (c SM) GetUint(key string, defVal uint) uint {
	return SV(c[key]).Uint(defVal)
}

func (c SM) GetInt64(key string, defVal int64) int64 {
	return SV(c[key]).Int64(defVal)
}

func (c SM) GetUint64(key string, defVal uint64) uint64 {
	return SV(c[key]).Uint64(defVal)
}

func (c SM) GetDuration(key string, defVal time.Duration) time.Duration {
	return SV(c[key]).Duration(defVal)
}

func (c SM) GetUnixTime(key string, defVal *time.Time) time.Time {
	return SV(c[key]).UnixTime(defVal)
}

func (c SM) GetBool(key string) bool {
	return SV(c[key]).Bool()
}

func (c SM) GetDataSize(key string, defVal int64) int64 {
	return SV(c[key]).DataSize(defVal)
}
