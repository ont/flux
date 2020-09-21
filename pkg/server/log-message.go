package server

import log "github.com/sirupsen/logrus"

var (
	HostFieldName    string // name of "host" field in json log message
	ProgramFieldName string // name of "program" field
	MessageFieldName string // name of "message" field
	RouteFieldName   string // name of "route" field
)

func init() {
	HostFieldName = GetenvStr("FLUX_HOST_FIELD_NAME", "HOST")
	ProgramFieldName = GetenvStr("FLUX_PROGRAM_FIELD_NAME", "PROGRAM")
	MessageFieldName = GetenvStr("FLUX_MESSAGE_FIELD_NAME", "MESSAGE")
	RouteFieldName = GetenvStr("FLUX_ROUTE_FIELD_NAME", "ROUTE")
}

type LogMessage map[string]interface{}

func (l LogMessage) Host() string {
	return l.getFieldStr(HostFieldName)
}

func (l LogMessage) Program() string {
	return l.getFieldStr(ProgramFieldName)
}

func (l LogMessage) Message() string {
	return l.getFieldStr(MessageFieldName)
}

func (l LogMessage) Route() string {
	return l.getFieldStr(RouteFieldName)
}

func (l LogMessage) Validate() bool {
	return l.hasField(HostFieldName) &&
		l.hasField(ProgramFieldName) &&
		l.hasField(MessageFieldName) &&
		l.hasField(RouteFieldName)
}

func (l LogMessage) getFieldStr(name string) string {
	if value, ok := l[name].(string); ok {
		return value
	} else {
		log.WithField("field_name", name).
			WithField("value", l[name]).
			Error("can't find/convert field from JSON to string")
		return ""
	}
}

func (l LogMessage) hasField(name string) bool {
	_, found := l[name]
	return found
}
