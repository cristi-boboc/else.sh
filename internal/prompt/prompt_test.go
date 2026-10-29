package prompt

import "testing"

func TestDecide(t *testing.T) {
	cmds := func() map[string]struct{} {
		return map[string]struct{}{"git": {}, "who": {}, "ls": {}, "docker": {}}
	}
	opts := Options{MinWords: 2, TypoGuard: true, Commands: cmds}
	tests := []struct {
		line   []string
		accept bool
	}{
		{[]string{"refactor", "the", "auth", "module"}, true},
		{[]string{"why", "is", "CI", "red"}, true}, // intent word beats "who"
		{[]string{"gti", "status"}, false},         // typo of git
		{[]string{"dokcer", "ps"}, false},          // transposition
		{[]string{"foo"}, false},                   // single word
		{[]string{"tests?"}, true},                 // question
		{[]string{"./build.sh", "now"}, false},     // path
		{[]string{"summarize"}, true},              // intent word alone
		{[]string{"lsd", "the", "files"}, false},   // len 3, one edit from ls? no: distance 1 -> typo
		{[]string{"translate", "this", "to", "go"}, true},
		{[]string{}, false},
	}
	for _, tt := range tests {
		if got := Decide(tt.line, opts); got.Accept != tt.accept {
			t.Errorf("Decide(%q) = %v (%s), want %v", tt.line, got.Accept, got.Reason, tt.accept)
		}
	}
}

func TestOSADistance(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want int
	}{{"gti", "git", 1}, {"git", "git", 0}, {"kitten", "sitting", 3}, {"ab", "ba", 1}} {
		if got := osaDistance(tt.a, tt.b); got != tt.want {
			t.Errorf("osaDistance(%q,%q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
