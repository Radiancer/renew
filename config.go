package main

import (
	"gopkg.in/yaml.v1"
	"io/ioutil"
)

type config struct {
	//正在运行的app名称，默认当前目录名
	AppName string `yaml:"appname"`
	//指定输出执行的程序路径
	Output string `yaml:"output"`
	// 需要添加watch文件后缀名，默认为'.go'
	WatchExts []string `yaml:"watch_exts"`
	// 需要添加watch目录，默认为当前文件夹
	WatchPaths []string `yaml:"watch_paths"`
	// 对于需要编译的包或文件，先使用-p参数
	BuildPkg string `yaml:"build_pkg"`
	//需要排除的路径。
	ExcludedPaths []string `yaml:"excluded_paths"`
	// 指定程序是否自动运行
	DisableRun bool `yaml:"disable_run"`
}

func parseYaml() *config {
	cfg := &config{}
	//将renew.yaml的内容解析到yamlInfo
	yamlInfo, err := ioutil.ReadFile(configYaml)
	if err != nil {
		panic(err)
	}
	//将yamlInfo 的信息对应写到cfg里面。
	if err := yaml.Unmarshal(yamlInfo, cfg); err != nil {
		panic(err)
	}
	return cfg
}
