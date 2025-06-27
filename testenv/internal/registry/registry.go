package registry

var registry = make(map[string]func() error)

func Register(name string, fn func() error) {
	registry[name] = fn
}

func Get(name string) (func() error, bool) {
	fn, ok := registry[name]
	return fn, ok
}

func All() map[string]func() error {
	return registry
}

