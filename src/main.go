package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

const (
	apexPath     = "/sys/class/apex"
	pollInterval = time.Second
	missingValue = "n/a"
)

type TpuStats struct {
	Index       int
	Path        string
	Framework   string
	Driver      string
	Temp        int
	Status      string
	Runtime     uint64
	Error       string
	TempRead    bool
	RuntimeRead bool
}

func getTpus(basePath string) ([]TpuStats, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("read TPU directory %q: %w", basePath, err)
	}

	tpuStats := make([]TpuStats, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "apex_") {
			continue
		}

		index, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), "apex_"))
		if err != nil {
			continue
		}

		path := filepath.Join(basePath, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat TPU device %q: %w", path, err)
		}
		if !info.IsDir() {
			continue
		}

		tpuStats = append(tpuStats, TpuStats{
			Index: index,
			Path:  path,
		})
	}

	sort.Slice(tpuStats, func(i, j int) bool {
		return tpuStats[i].Index < tpuStats[j].Index
	})

	return tpuStats, nil
}

func headers() []string {
	return []string{"Name", "Framework", "Driver", "Temp", "Status", "Runtime", "Error"}
}

func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readTpuStats(tpu TpuStats) TpuStats {
	var readErrors []string

	readString := func(name string, destination *string) {
		value, err := readSysfsFile(filepath.Join(tpu.Path, name))
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: %v", name, err))
			*destination = missingValue
			return
		}
		*destination = value
	}

	readInt := func(name string, destination *int) {
		value, err := readSysfsFile(filepath.Join(tpu.Path, name))
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: %v", name, err))
			return
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: invalid integer %q", name, value))
			return
		}
		*destination = parsed
		if name == "temp" {
			tpu.TempRead = true
		}
	}

	readUint := func(name string, destination *uint64) {
		value, err := readSysfsFile(filepath.Join(tpu.Path, name))
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: %v", name, err))
			return
		}
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: invalid integer %q", name, value))
			return
		}
		*destination = parsed
		tpu.RuntimeRead = true
	}

	readString("framework_version", &tpu.Framework)
	readString("driver_version", &tpu.Driver)
	readInt("temp", &tpu.Temp)
	readString("status", &tpu.Status)
	readUint(filepath.Join("power", "runtime_active_time"), &tpu.Runtime)

	tpu.Error = strings.Join(readErrors, "; ")
	return tpu
}

func formatTemperature(millicelsius int) string {
	return fmt.Sprintf("%.1f°C", float64(millicelsius)/1000)
}

func formatRuntime(microseconds uint64) string {
	if microseconds > uint64((int64(1<<63-1) / int64(time.Microsecond))) {
		return fmt.Sprintf("%dµs", microseconds)
	}
	return time.Duration(microseconds * uint64(time.Microsecond)).String()
}

func tableData(tpus []TpuStats) pterm.TableData {
	data := pterm.TableData{headers()}
	for _, tpu := range tpus {
		temp := missingValue
		if tpu.TempRead {
			temp = formatTemperature(tpu.Temp)
		}

		runtime := missingValue
		if tpu.RuntimeRead {
			runtime = formatRuntime(tpu.Runtime)
		}

		data = append(data, []string{
			fmt.Sprintf("apex_%d", tpu.Index),
			tpu.Framework,
			tpu.Driver,
			temp,
			tpu.Status,
			runtime,
			tpu.Error,
		})
	}
	return data
}

func main() {
	pterm.Info.Println("Reading data from " + apexPath)

	tpus, err := getTpus(apexPath)
	if err != nil {
		pterm.Error.Println(err)
		return
	}
	if len(tpus) == 0 {
		pterm.Warning.Println("No TPUs found")
		return
	}

	stop := make(chan struct{})
	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n')
		close(stop)
	}()

	area, err := pterm.DefaultArea.Start()
	if err != nil {
		pterm.Error.Println("start terminal area:", err)
		return
	}
	defer area.Stop()

	for {
		select {
		case <-stop:
			return
		default:
		}

		currentTpus, err := getTpus(apexPath)
		if err != nil {
			pterm.Error.Println(err)
			return
		}

		if len(currentTpus) == 0 {
			area.Update("No TPUs found")
		} else {
			for i := range currentTpus {
				currentTpus[i] = readTpuStats(currentTpus[i])
			}
			table, err := pterm.DefaultTable.WithHasHeader().WithData(tableData(currentTpus)).Srender()
			if err != nil {
				pterm.Error.Println("render TPU table:", err)
				return
			}
			area.Update(table)
		}

		time.Sleep(pollInterval)
	}
}
