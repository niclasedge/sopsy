package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"testing"
	"time"
)

// helperArg as the first argument turns the test binary into a small helper
// program that tests start as a child process.
const helperArg = "__helper__"

// TestMain lets the test binary act as sopsy itself (SOPSY_TEST_MAIN=sopsy)
// or as a helper child, so process tests need no separate build.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == helperArg {
		os.Exit(helper(os.Args[2:]))
	}
	if os.Getenv("SOPSY_TEST_MAIN") == "sopsy" {
		os.Exit(Main())
	}
	os.Exit(m.Run())
}

func helper(args []string) int {
	switch args[0] {
	case "printenv":
		v, ok := os.LookupEnv(args[1])
		if !ok {
			return 1
		}
		fmt.Print(v)
		return 0
	case "args":
		for _, a := range args[1:] {
			fmt.Printf("%q\n", a)
		}
		return 0
	case "exit":
		code, _ := strconv.Atoi(args[1])
		return code
	case "marker":
		_ = os.WriteFile(args[1], nil, 0o600)
		return 0
	case "trap":
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		fmt.Println("ready")
		select {
		case <-ch:
			_ = os.WriteFile(args[1], []byte("interrupted"), 0o600)
			return 42
		case <-time.After(20 * time.Second):
			return 99
		}
	}
	return 2
}

// helperCmd returns the argv that runs the helper in the given mode.
func helperCmd(args ...string) []string {
	self, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return append([]string{self, helperArg}, args...)
}
