package main

import (
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
)

var (
	cfg         *config
	output      string
	buildPkg    string
	showVersion bool
	showHelp    bool
	currpath    string

	started chan bool
)
var configYaml = "renew.yaml"

func init() {
	flag.StringVar(&output, "o", "", "go build output")
	flag.StringVar(&buildPkg, "p", "", "go build packages")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.BoolVar(&showHelp, "h", false, "show help")
	flag.Parse()
	logrus.SetReportCaller(true)
}
func main() {
	fmt.Printf(`
 ________  _______   ________   _______   ___       __      
|\   __  \|\  ___ \ |\   ___  \|\  ___ \ |\  \     |\  \    
\ \  \|\  \ \   __/|\ \  \\ \  \ \   __/|\ \  \    \ \  \   
 \ \   _  _\ \  \_|/_\ \  \\ \  \ \  \_|/_\ \  \  __\ \  \  
  \ \  \\  \\ \  \_|\ \ \  \\ \  \ \  \_|\ \ \  \|\__\_\  \ 
   \ \__\\ _\\ \_______\ \__\\ \__\ \_______\ \____________\
    \|__|\|__|\|_______|\|__| \|__|\|_______|\|____________|  built with Golang

`)
	/*
		watchPrint()
		buildingPrint()
		runingPrint()*/

	// renew -h 输出的帮助信息
	if showHelp {
		fmt.Println("Usage of renew:\n\nIf no command is provided renew will start the runner with the provided flags\n\nCommands:\n  init  creates a renew.toml file with default settings to the current directory\n\nFlags:")
		flag.PrintDefaults()
		os.Exit(0)
	}
	//renew -v 查看版本号
	if showVersion {
		printVersion()
	}
	//获取当前工作目录对应的根路径。
	currpath, _ = os.Getwd()
	//解析renew.yaml文件。
	cfg := parseYaml()

	//配置cfg里面的信息

	//配置Appname
	if cfg.AppName == "" {
		cfg.AppName = path.Base(currpath)
	}

	//配置output
	if output != "" {
		cfg.Output = output
	}

	if cfg.Output == "" {
		if runtime.GOOS == "windows" {
			cfg.Output = "./" + cfg.AppName + ".exe"
		}
	}

	run(currpath)
}
