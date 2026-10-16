package websocket

type ServerOption func(opt *serverOption)

type serverOption struct {
	Authentication
	pattern string
}

func newOption(opts ...ServerOption) serverOption { //ServerOption是用来改变serverOption内部属性的函数
	o := serverOption{
		Authentication: new(authentication),
		pattern:        "/ws",
	}
	for _, opt := range opts { //取出修改函数，用来修改serverOption的内部参数
		opt(&o)
	}
	return o
}

// WithAuthentication 返回一个用来修改serverOption的Authentication的函数
func WithAuthentication(authentication Authentication) ServerOption {
	return func(opt *serverOption) {
		opt.Authentication = authentication
	}
}

// WithHandlerPatten 返回一个用来修改serverOption的pattern的函数
func WithHandlerPatten(pattern string) ServerOption {
	return func(opt *serverOption) {
		opt.pattern = pattern
	}
}
