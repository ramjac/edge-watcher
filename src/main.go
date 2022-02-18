package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

type TpuStats struct {
	Index     int
	Path      string
	Framework string
	Driver    string
	Temp      int
	Status    string
	Runtime   int // doesn't appear to work
}

func getTpus(basePath string) (tpuStats []TpuStats) {
	err := filepath.Walk(basePath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			tmp := info.Name()[strings.LastIndex(info.Name(), "_")+1:]
			idx, _ := strconv.Atoi(string(tmp))
			tpuStats = append(tpuStats, TpuStats{Index: idx, Path: info.Name()})
		}

		return nil
	})
	if err != nil {
		pterm.Error.Printf("error walking the path %q: %v\n", basePath, err)
		return
	}

	return tpuStats
}

func headers() []string {
	return []string{"Name", "Framework", "Driver", "Temp", "Status", "Runtime"}
}

func main() {
	apexPath := "/sys/class/apex/"
	pterm.Info.Println("Reading data from " + apexPath)

	tpuStats := getTpus(apexPath)

	if len(tpuStats) < 1 {
		pterm.Warning.Println("No TPUs found")
		return
	}

	ch := make(chan string)
	go func(ch chan string) {
		reader := bufio.NewReader(os.Stdin)
		for {
			s, _ := reader.ReadString('\n')
			ch <- s
		}
	}(ch)

	sort.Slice(tpuStats, func(i, j int) bool {
		return tpuStats[i].Index < tpuStats[j].Index
	})

	area, _ := pterm.DefaultArea.Start() // Start the Area printer, with the Center option.
	for {
		select {
		case <-ch:
			area.Stop()
			return
		default:

			tableData := pterm.TableData{headers()}

			for _, tpu := range tpuStats {
				framework, _ := exec.Command("cat", "/sys/class/apex/apex_0/framework_version").Output()
				tpu.Framework = strings.TrimSpace(string(framework))
				driver, _ := exec.Command("cat", "/sys/class/apex/apex_0/driver_version").Output()
				tpu.Driver = strings.TrimSpace(string(driver))
				temp, _ := exec.Command("cat", "/sys/class/apex/apex_0/temp").Output()
				tpu.Temp, _ = strconv.Atoi(strings.TrimSpace(string(temp)))
				statusOut, _ := exec.Command("cat", "/sys/class/apex/apex_0/status").Output()
				tpu.Status = strings.TrimSpace(string(statusOut))
				runtime, _ := exec.Command("cat", "/sys/class/apex/apex_0/power/runtime_active_time").Output()
				tpu.Runtime, _ = strconv.Atoi(strings.TrimSpace(string(runtime)))

				tableData = append(tableData, []string{strconv.Itoa(tpu.Index), tpu.Framework, tpu.Driver, strconv.Itoa(tpu.Temp), tpu.Status, strconv.Itoa(tpu.Runtime)})
			}

			table, _ := pterm.DefaultTable.WithHasHeader().WithData(tableData).Srender()

			area.Update(table)
			time.Sleep(time.Second)
		}
	}
}
