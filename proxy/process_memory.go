package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ProcessMemory struct {
	RAMBytes  int64
	VRAMBytes int64
}

func (p *Process) GetMemoryUsage() (ProcessMemory, error) {
	var result ProcessMemory

	if p.cmd == nil || p.cmd.Process == nil {
		return result, nil
	}

	pid := p.cmd.Process.Pid

	ram, err := getProcessRSS(pid)
	if err != nil {
		p.proxyLogger.Debugf("<%s> could not read RSS: %v", p.ID, err)
	} else {
		result.RAMBytes = ram
	}

	vram, err := getProcessVRAM(pid)
	if err != nil {
		p.proxyLogger.Debugf("<%s> could not read VRAM: %v", p.ID, err)
	} else {
		result.VRAMBytes = vram
	}

	return result, nil
}

func getProcessRSS(pid int) (int64, error) {
	file, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(file)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			parts := strings.Fields(strings.TrimPrefix(line, "VmRSS:"))
			if len(parts) >= 2 {
				kb, err := strconv.ParseInt(parts[0], 10, 64)
				if err != nil {
					return 0, err
				}
				return kb * 1024, nil
			}
		}
	}
	return 0, fmt.Errorf("VmRSS not found")
}

func getProcessVRAM(pid int) (int64, error) {
	cmd := exec.Command("nvidia-smi",
		"--query-compute-apps=pid,used_memory",
		"--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	pidStr := fmt.Sprintf("%d", pid)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == pidStr {
			memStr := strings.TrimSpace(parts[1])
			memStr = strings.TrimSpace(strings.TrimSuffix(memStr, " MiB"))
			miB, err := strconv.ParseInt(memStr, 10, 64)
			if err != nil {
				continue
			}
			return miB * 1024 * 1024, nil
		}
	}
	return 0, fmt.Errorf("pid %d not found in nvidia-smi output", pid)
}
