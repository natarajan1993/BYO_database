package utils

func Assert(b bool, message string) {
	if b {
		panic(message)
	}
}
