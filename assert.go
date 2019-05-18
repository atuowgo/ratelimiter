package ratelimiter

type AssertError struct {
	Message string
}

func (e *AssertError) Error() string {
	return "assert fail: " + e.Message
}

func NotEmpty(str, message string) {
	if str == "" {
		panic(AssertError{Message: message})
	}
}

func IsTrue(value bool, message string) {
	if !value {
		panic(AssertError{Message: message})
	}
}

func NotNil(obj interface{}, message string) {
	if obj == nil {
		panic(AssertError{Message: message})
	}
}
