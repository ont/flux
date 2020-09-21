package server

import (
	"github.com/dop251/goja"
	log "github.com/sirupsen/logrus"
)

type PointTags map[string]string
type PointValues map[string]interface{}

type Script struct {
	Tags   PointTags
	Values PointValues
	Data   PointValues

	vm      *goja.Runtime
	program *goja.Program
}

func NewScript(source string) *Script {
	program, err := goja.Compile("script", source, false)
	if err != nil {
		log.WithError(err).Fatal("Can't compile script")
	}

	return &Script{
		Tags:   make(PointTags),
		Values: make(PointValues),
		Data:   make(PointValues),

		vm:      goja.New(),
		program: program,
	}
}

func (s *Script) Process(message string) error {
	s.vm.Set("tags", s.Tags)
	s.vm.Set("values", s.Values)
	s.vm.Set("data", s.Data)
	s.vm.Set("message", message)

	_, err := s.vm.RunProgram(s.program)
	return err
}
