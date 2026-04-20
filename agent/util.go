package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

type SystemInfo struct {
	OS          string `json:"os"`
	Hostname    string `json:"hostname"`
	Username    string `json:"username"`
	CurrentDir  string `json:"currentDir"`
	CPUUsage    int    `json:"cpuUsage"` // Percentage 0-100
	RAMUsed     int    `json:"ramUsed"`  // MB
	RAMTotal    int    `json:"ramTotal"` // MB
	RAMUsage    int    `json:"ramUsage"` // Percentage 0-100
}

func getSystemInfo() SystemInfo {
	info := SystemInfo{}

	hostname, _ := os.Hostname()
	info.Hostname = hostname

	// Fallback environment variables
	info.Username = os.Getenv("USERNAME")
	if info.Username == "" {
		info.Username = os.Getenv("USER")
	}

	cwd, _ := os.Getwd()
	info.CurrentDir = cwd

	// PowerShell to get actual hardware stats
	// Gets OS version, CPU total load, RAM total, RAM free
	psScript := `
$os = (Get-WmiObject Win32_OperatingSystem).Caption
$cpu = (Get-WmiObject Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average
$cs = Get-WmiObject Win32_ComputerSystem
$ramTotal = [math]::Round($cs.TotalPhysicalMemory / 1MB)
$osstats = Get-WmiObject Win32_OperatingSystem
$ramFree = [math]::Round($osstats.FreePhysicalMemory / 1KB)
Write-Output "$os|!|$cpu|!|$ramTotal|!|$ramFree"
`
	cmd := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-Command", psScript)
	outputBytes, err := cmd.Output()
	if err == nil {
		output := strings.TrimSpace(string(outputBytes))
		
		// Some output might contain ANSI escape codes
		ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		cleanOutput := ansiRegex.ReplaceAllString(output, "")
		
		parts := strings.Split(cleanOutput, "|!|")
		if len(parts) == 4 {
			info.OS = strings.TrimSpace(parts[0])
			
			cpuLoad, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			info.CPUUsage = int(cpuLoad)
			
			ramTot, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
			ramFr, _ := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
			
			info.RAMTotal = int(ramTot)
			if info.RAMTotal > 0 {
				info.RAMUsed = info.RAMTotal - int(ramFr)
				info.RAMUsage = int((float64(info.RAMUsed) / float64(info.RAMTotal)) * 100)
			}
		}
	} else {
		info.OS = "Windows (Hardware stats unavailable without pwsh)"
	}

	return info
}
