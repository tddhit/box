package metrics

var defaultOption = options{
	useQPS: false,
	useEPS: false,
	useTPQ: false,
}

type options struct {
	useQPS bool
	useEPS bool
	useTPQ bool
}

type Option func(*options)

func WithQPS() Option {
	return func(o *options) {
		o.useQPS = true
	}
}

func WithEPS() Option {
	return func(o *options) {
		o.useEPS = true
	}
}

func WithTPQ() Option {
	return func(o *options) {
		o.useTPQ = true
	}
}
