package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	// 명령행 인자가 부족한 경우 사용법 출력
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  To check if a port is open: port-checker <port>")
		fmt.Println("  To kill a local port: port-checker -kill <port>")
		return
	}

	// 첫 번째 인자로 옵션을 결정하고 실행
	switch option := os.Args[1]; option {
	case "-kill":
		// '-kill' 옵션인 경우 포트 번호가 함께 제공되었는지 확인
		if len(os.Args) != 3 {
			fmt.Println("Usage: port-checker -kill <port>")
			return
		}
		portStr := os.Args[2]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Println("Invalid port number:", portStr)
			return
		}
		// 로컬 포트 종료 함수 호출
		err = killLocalPort(port)
		if err != nil {
			fmt.Printf("Failed to kill port %d: %v\n", port, err)
		} else {
			fmt.Printf("Port %d successfully killed\n", port)
		}
	default:
		// 포트 번호가 주어진 경우 포트 상태 확인
		portStr := os.Args[1]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Println("Invalid port number:", portStr)
			return
		}
		// 포트 상태 확인 함수 호출
		err = checkPort(port)
		if err != nil {
			fmt.Printf("Port %d is closed\n", port)
		} else {
			// 포트가 열려 있으면 해당 프로세스 이름 가져오기 시도
			processName, err := getProcessName(port)
			if err != nil {
				fmt.Printf("Failed to retrieve process information for port %d: %v\n", port, err)
			} else {
				fmt.Printf("Port %d is open.\n", port)
				fmt.Printf("Process Name: %s\n", processName)
			}
		}
	}

}

// checkPort는 주어진 포트가 열려 있는지 확인합니다.
func checkPort(port int) error {
	address := fmt.Sprintf("localhost:%d", port)
	_, err := net.Dial("tcp", address)
	return err
}

// killLocalPort는 주어진 포트를 사용 중인 프로세스를 종료합니다.
func killLocalPort(port int) error {
	pid, err := findProcessID(port)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/PID", pid)
	case "darwin", "linux":
		cmd = exec.Command("kill", "-9", pid)
	default:
		return fmt.Errorf("Unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Run()
}

// findProcessID는 주어진 포트를 사용 중인 프로세스의 PID를 찾습니다.
func findProcessID(port int) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("netstat", "-aon")
	case "darwin", "linux":
		cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	default:
		return "", fmt.Errorf("Unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(":%d", port)) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// PID는 마지막 필드에 있을 것으로 가정
				pidField := fields[len(fields)-1]
				// 숫자인지 확인하여 PID인지 확인
				if _, err := strconv.Atoi(pidField); err == nil {
					return pidField, nil
				}
			}
		}
	}

	return "", fmt.Errorf("Process ID not found for port %d", port)
}

// getProcessName는 주어진 포트를 사용 중인 프로세스의 이름을 가져옵니다.
func getProcessName(port int) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("netstat", "-aon")
	case "darwin", "linux":
		cmd = exec.Command("lsof", "-i", fmt.Sprintf(":%d", port))
	default:
		return "", fmt.Errorf("Unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(":%d", port)) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				pid := fields[len(fields)-1]
				// PID를 기반으로 프로세스 이름 가져오기 시도
				processName, err := getProcessNameByPID(pid)
				if err != nil {
					return "", err
				}
				return processName, nil
			}
		}
	}
	return "", fmt.Errorf("Process information not found for port %d", port)
}

// getProcessNameByPID는 주어진 PID에 해당하는 프로세스의 이름을 가져옵니다.
func getProcessNameByPID(pid string) (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %s", pid), "/FO", "CSV", "/NH")
	case "darwin", "linux":
		cmd = exec.Command("ps", "-p", pid, "-o", "comm=")
	default:
		return "", fmt.Errorf("Unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// OS에 따라 프로세스 이름 가져오기
	var processName string
	switch runtime.GOOS {
	case "windows":
		// tasklist 명령어의 CSV 출력 파싱
		csvOutput := strings.TrimSpace(string(output))
		if csvOutput == "" {
			return "", fmt.Errorf("Process name not found for PID: %s", pid)
		}
		fields := strings.Split(csvOutput, "\",\"")
		if len(fields) >= 2 {
			processName = strings.Trim(fields[0], "\"")
		} else {
			return "", fmt.Errorf("Unexpected tasklist output format for PID: %s", pid)
		}
	case "darwin", "linux":
		// ps 명령어 출력에서 직접 프로세스 이름 추출
		processName = strings.TrimSpace(string(output))
		if processName == "" {
			return "", fmt.Errorf("Process name not found for PID: %s", pid)
		}
	default:
		return "", fmt.Errorf("Unsupported operating system: %s", runtime.GOOS)
	}

	return processName, nil
}
