package server

import (
	"io"
	"regexp"
	"strings"

	"github.com/alecthomas/participle"
	"github.com/mohae/deepcopy"
	log "github.com/sirupsen/logrus"
)

type Grammar struct {
	Routes []*Route `{ @@ }`
}

type Route struct {
	Name    string    `"route" @String "{"`
	Metrics []*Metric `{ @@ } "}"`
}

type Metric struct {
	Name   string   `"metric" @String "{"`
	Params []*Param `{ @@  } "}"`

	parseJSON bool
	re        *regexp.Regexp
	script    *Script
	eventName string // name of event, i.e. name of special column in influx which will contain "1" value
}

type Param struct {
	Key   string `@Ident "="`
	Value string `@(RawString|String|Ident)`
}

func NewGrammar(reader io.Reader) *Grammar {
	var grammar Grammar

	parser := participle.MustBuild(&Grammar{}, nil)
	parser.Parse(reader, &grammar)

	return &grammar
}

func (m *Metric) unpackParams() {
	reStr := m.Get("regexp")
	format := m.Get("format")

	if reStr == "" && format == "" {
		log.Fatal("empty or missed 'regexp' or 'format' fields in metric")
	}

	if script := m.Get("script"); script != "" {
		m.script = NewScript(script)
	}

	if reStr != "" {
		m.re = regexp.MustCompile(reStr)

		m.eventName = m.Get("event")

	} else {
		if format != "json" {
			log.Fatalf("unsupported format: %s", format)
		}

		if m.script == nil {
			log.Fatal("missing 'script' section; 'script' must be used for specifying tags and values from json")
		}

		m.parseJSON = true
	}
}

func (m *Metric) Clone() *Metric {
	clone, ok := deepcopy.Copy(m).(*Metric)
	if !ok {
		log.Fatal("can't do deepclone for metric")
	}

	clone.unpackParams()

	return clone
}

func (m *Metric) Get(param string) string {
	for _, pobj := range m.Params {
		if pobj.Key == param {
			return pobj.Value
		}
	}

	return ""
}

func (m *Metric) GetBool(param string) bool {
	for _, pobj := range m.Params {
		if pobj.Key == param {
			return strings.TrimSpace(strings.ToLower(pobj.Value)) == "true"
		}
	}

	return false
}
