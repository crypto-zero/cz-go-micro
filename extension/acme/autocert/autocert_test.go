package autocert

import (
	"testing"
)

func TestAutocert(t *testing.T) {
	l := NewProvider()
	if _, ok := l.(*autoCertProvider); !ok {
		t.Error("NewProvider() didn't return an autoCertProvider")
	}
	// TODO: Travis CI doesn't let us bind :443
	// if _, err := l.NewListener(); err != nil {
	// 	t.Error(err.Error())
	// }
}
