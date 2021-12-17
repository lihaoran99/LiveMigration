package helper

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

//CheckError 处理致命错误并结束程序
func CheckError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

//CloseFileSafely 延迟关闭文件并且处理错误
func CloseFileSafely(file *os.File) {
	err := file.Close()
	CheckError(err)
}

//ChangeWorkDir2ExecDir 用于改变工作目录为可执行程序所在的目录
func ChangeWorkDir2ExecDir() {
	workDir, _ := os.Getwd()
	execDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	if workDir != execDir {
		_ = os.Chdir(execDir)
	}
}

//PrintUsageAndExit 用于打印程序的用法
func PrintUsageAndExit() {
	fmt.Println("Only Support One Option At An Execution!!!")
	fmt.Println("Usage: ")
	fmt.Println("    ./LiveMigration --list")
	fmt.Println("    ./LiveMigration -l")
	fmt.Println("    ./LiveMigration --realtime")
	fmt.Println("    ./LiveMigration -r")
	fmt.Println("    ./FLiveMigration --migrate vmname hostname")
	fmt.Println("    ./LiveMigration -m vmname1 hostname1 (vmname2 hostname2 (...) )")
	fmt.Println("    ./LiveMigration --multigrate vmname1 hostname1 (vmname2 hostname2 (...) )")
	fmt.Println("    ./LiveMigration --heuristic")
	fmt.Println("    ./LiveMigration --shuffle")
	fmt.Println("    ./LiveMigration -s")
	fmt.Println("Example: ")
	fmt.Println("    ./LiveMigration --list")
	fmt.Println("    ./LiveMigration -l")
	fmt.Println("    ./LiveMigration --realtime")
	fmt.Println("    ./LiveMigration -r")
	fmt.Println("    ./LiveMigration --migrate RabbitMQ-4 CNA01")
	fmt.Println("    ./LiveMigration -m RabbitMQ-4 CNA01")
	fmt.Println("    ./LiveMigration -m RabbitMQ-4 CNA01 RabbitMQ-5 CNA02")
	fmt.Println("    ./LiveMigration --multigrate RabbitMQ-4 CNA01 RabbitMQ-5 CNA02")
	fmt.Println("    ./LiveMigration --heuristic")
	fmt.Println("    ./LiveMigration --shuffle")
	fmt.Println("    ./LiveMigration -s")
	os.Exit(0)
}