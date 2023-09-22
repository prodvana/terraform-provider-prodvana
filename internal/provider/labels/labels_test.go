package labels

import "testing"

func TestLabelValueRegex(t *testing.T) {
	valid := []string{
		"",
		"foo",
		"foo-bar",
		"foo.bar",
		"foo\\bar",
		"foo_bar",
		"foo+bar",
		"foo@bar",
		"foo.bar-baz\\qux+quux@corge",
		"-foo-",
		".foo.",
		"\\foo\\",
		"_foo_",
		"+foo+",
		"@foo@",
	}

	invalid := []string{
		"foo bar",
		"foo/bar",
		"foo$bar",
		"foo%bar",
		"foo^bar",
		"foo&bar",
		"foo*bar",
		"foo(bar",
		"foo)bar",
		"foo=bar",
		"foo{bar",
		"foo}bar",
		"foo[bar",
		"foo]bar",
		"foo|bar",
		"foo:bar",
		"foo;bar",
		"foo'bar",
		"foo\"bar",
		"foo<bar",
		"foo>bar",
		"foo,bar",
		"foo?bar",
		"foo!bar",
	}

	for _, v := range valid {
		if !labelValueRegex.MatchString(v) {
			t.Errorf("expected %q to match regex", v)
		}
	}

	for _, v := range invalid {
		if labelValueRegex.MatchString(v) {
			t.Errorf("expected %q to not match regex", v)
		}
	}
}
