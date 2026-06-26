package pkg

import "testing"

func TestLauncherScriptContentEmbedsLunexBin(t *testing.T) {
	script := launcherScriptContent("/opt/luna-pm/main.lx", "/data/data/com.termux/files/home/lunex-lang/lunex")
	if !contains(script, `LUNEXBIN="/data/data/com.termux/files/home/lunex-lang/lunex"`) {
		t.Fatalf("script missing lunex bin assignment: %q", script)
	}
	if !contains(script, `exec "$LUNEXBIN" run "/opt/luna-pm/main.lx" "$@"`) {
		t.Fatalf("script missing exec line: %q", script)
	}
}

func TestParseRunOptionsSeparatesScriptArgs(t *testing.T) {
	emit, scriptArgs, err := parseRunOptions([]string{"--emit", "ast", "install", "pkg"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emit != emitModeAST {
		t.Fatalf("expected emit ast, got %q", emit)
	}
	if len(scriptArgs) != 2 || scriptArgs[0] != "install" || scriptArgs[1] != "pkg" {
		t.Fatalf("unexpected script args: %#v", scriptArgs)
	}
}

func TestParseRunOptionsAllowsDashArgsForScripts(t *testing.T) {
	emit, scriptArgs, err := parseRunOptions([]string{"-v", "install"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emit != "" {
		t.Fatalf("expected no emit mode, got %q", emit)
	}
	if len(scriptArgs) != 2 || scriptArgs[0] != "-v" || scriptArgs[1] != "install" {
		t.Fatalf("unexpected script args: %#v", scriptArgs)
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	return indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
