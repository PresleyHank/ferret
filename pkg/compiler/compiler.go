package compiler

import (
	"regexp"
	"strings"

	"github.com/MontFerret/ferret/pkg/parser"
	"github.com/MontFerret/ferret/pkg/runtime"
	"github.com/MontFerret/ferret/pkg/runtime/core"
	"github.com/MontFerret/ferret/pkg/stdlib"
	"github.com/pkg/errors"
)

var fnNameValidation = regexp.MustCompile("^[a-zA-Z]+[a-zA-Z0-9_]*(::[a-zA-Z]+[a-zA-Z0-9_]*)*$")

type FqlCompiler struct {
	funcs map[string]core.Function
}

func New(setters ...Option) *FqlCompiler {
	c := &FqlCompiler{}
	opts := &Options{}

	for _, setter := range setters {
		setter(opts)
	}

	if !opts.noStdlib {
		c.funcs = stdlib.NewLib()
	} else {
		c.funcs = make(map[string]core.Function)
	}

	return c
}

func (c *FqlCompiler) RegisterFunction(name string, fun core.Function) error {
	_, exists := c.funcs[name]

	if exists {
		return errors.Errorf("function already exists: %s", name)
	}

	// validation the name
	if !fnNameValidation.MatchString(name) {
		return errors.Errorf("invalid function name: %s", name)
	}

	c.funcs[strings.ToUpper(name)] = fun

	return nil
}

func (c *FqlCompiler) RemoveFunction(name string) {
	delete(c.funcs, strings.ToUpper(name))
}

func (c *FqlCompiler) RegisterFunctions(funcs map[string]core.Function) error {
	for name, fun := range funcs {
		if err := c.RegisterFunction(name, fun); err != nil {
			return err
		}
	}

	return nil
}

func (c *FqlCompiler) RegisteredFunctions() []string {
	res := make([]string, 0, len(c.funcs))

	for k := range c.funcs {
		res = append(res, k)
	}

	return res
}

func (c *FqlCompiler) RegisteredFunctionsNS(namespace string) []string {
	res := make([]string, 0, len(c.funcs))

	for k := range c.funcs {
		if strings.HasPrefix(k, namespace) {
			res = append(res, k)
		}
	}

	return res
}

func (c *FqlCompiler) Compile(query string) (program *runtime.Program, err error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}

	defer func() {
		if r := recover(); r != nil {
			// find out exactly what the error was and set err
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}

			program = nil
		}
	}()

	p := parser.New(query)
	p.AddErrorListener(&errorListener{})

	l := newVisitor(query, c.funcs)

	res := p.Visit(l).(*result)

	if res.Ok() {
		program = res.Data().(*runtime.Program)
	} else {
		err = res.Error()
	}

	return program, err
}

func (c *FqlCompiler) MustCompile(query string) *runtime.Program {
	program, err := c.Compile(query)

	if err != nil {
		panic(err)
	}

	return program
}
