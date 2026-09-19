package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetTpusOnlyReturnsApexDirectoriesInIndexOrder(t *testing.T) {
	basePath := t.TempDir()
	for _, name := range []string{"apex_2", "apex_0", "not-a-tpu", "temp"} {
		if name == "temp" {
			if err := os.WriteFile(filepath.Join(basePath, name), []byte("42000\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.Mkdir(filepath.Join(basePath, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(basePath, "apex_not-an-index"), 0o700); err != nil {
		t.Fatal(err)
	}

	tpus, err := getTpus(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(tpus) != 2 {
		t.Fatalf("got %d TPUs, want 2", len(tpus))
	}
	if tpus[0].Index != 0 || tpus[1].Index != 2 {
		t.Fatalf("got indexes %d and %d, want 0 and 2", tpus[0].Index, tpus[1].Index)
	}
}

func TestReadTpuStatsReadsDeviceSpecificValuesAndReportsErrors(t *testing.T) {
	devicePath := t.TempDir()
	files := map[string]string{
		"framework_version":         "1.0\n",
		"driver_version":            "1.2\n",
		"temp":                      "87500\n",
		"status":                    "ALIVE\n",
		"power/runtime_active_time": "1234567\n",
	}
	for name, contents := range files {
		path := filepath.Join(devicePath, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	stats := readTpuStats(TpuStats{Index: 3, Path: devicePath})
	if stats.Framework != "1.0" || stats.Driver != "1.2" || stats.Temp != 87500 || stats.Status != "ALIVE" {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.Runtime != 1234567 {
		t.Fatalf("got runtime %d, want 1234567", stats.Runtime)
	}
	if stats.Error != "" {
		t.Fatalf("unexpected read error: %s", stats.Error)
	}

	if err := os.Remove(filepath.Join(devicePath, "status")); err != nil {
		t.Fatal(err)
	}
	stats = readTpuStats(TpuStats{Index: 3, Path: devicePath})
	if stats.Status != missingValue || stats.Error == "" {
		t.Fatalf("expected missing status to be reported: %+v", stats)
	}
}

func TestFormatting(t *testing.T) {
	if got := formatTemperature(87500); got != "87.5°C" {
		t.Fatalf("got temperature %q, want 87.5°C", got)
	}
	if got := formatRuntime(1234567); got != "1.234567s" {
		t.Fatalf("got runtime %q, want 1.234567s", got)
	}
}
