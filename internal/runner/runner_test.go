package runner

import (
	"errors"
	"slices"
	"testing"
)

func TestMergeEnvOverrides(t *testing.T) {
	got := MergeEnv([]string{"A=1", "TOKEN=old", "B=2"}, []string{"TOKEN=new"}, false)
	want := []string{"A=1", "B=2", "TOKEN=new"}
	if !slices.Equal(got, want) {
		t.Fatalf("MergeEnv() = %v, want %v", got, want)
	}
}

func TestMergeEnvCaseSensitiveOnUnix(t *testing.T) {
	got := MergeEnv([]string{"token=lower"}, []string{"TOKEN=new"}, false)
	want := []string{"token=lower", "TOKEN=new"}
	if !slices.Equal(got, want) {
		t.Fatalf("MergeEnv() = %v, want %v", got, want)
	}
}

func TestMergeEnvWindowsCasing(t *testing.T) {
	got := MergeEnv([]string{"=C:=C:\\work", "Path=C:\\bin", "Token=old"}, []string{"TOKEN=new"}, true)
	want := []string{"=C:=C:\\work", "Path=C:\\bin", "TOKEN=new"}
	if !slices.Equal(got, want) {
		t.Fatalf("MergeEnv() = %v, want %v", got, want)
	}
}

func TestRunNotFound(t *testing.T) {
	_, err := Run(Command{Args: []string{"sopsy-no-such-command"}})
	var se *StartError
	if !errors.As(err, &se) || se.Code != ExitNotFound {
		t.Fatalf("want StartError 127, got %v", err)
	}
}
