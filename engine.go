package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-ps"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	cmd       *exec.Cmd
	eventTime = make(map[string]int64)
	schTime   time.Time
)

var (
	red = "red"
	//blue    = "blue"
	//magenta = "magenta"
	//green   = "green"
	//cyan    = "cyan"
	yellow = "yellow"
)

func run(curpath string) {
	var paths []string
	//遍历整个目录，把所有没有需要监听变动的文件找出来
	collectFile(curpath, &paths)
	//对这些目录进行监听如果发生变动，就重新启动
	files := []string{}
	if buildPkg != "" {
		files = strings.Split(buildPkg, ",")
	}
	newWatcher(paths, files)

}

func newWatcher(paths []string, files []string) {
	//创建监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Errorf("watcher building error[%s]", err)
	}

	//监听器添加要监听的路径。

	for _, pth := range paths {
		err = watcher.Add(pth)
		if err != nil {
			logrus.Errorf("Fail to watch directory [ %s ]\n", err)
			os.Exit(2)
		}
	}
	//无限循环，不断监听文件的变化，并作出应答
	go func() {
		for {
			select {
			case e := <-watcher.Events:
				isbuild := true

				if !InWatchExt(e.Name) {
					continue
				}
				mt := getLastTime(e.Name)

				if t := eventTime[e.Name]; mt == t {
					logrus.Infof("not updated, skip")
					isbuild = false
				} else {
					colorPrint(red, e.Name+"file changes")
				}
				eventTime[e.Name] = mt
				if isbuild {
					go func() {
						//等待一秒直到没有文件更新。
						schTime = time.Now().Add(1 * time.Second)
						for {
							time.Sleep(time.Until(schTime))
							if time.Now().After(schTime) {
								break
							}
						}

						Autobuild(files)
					}()
				}
			case err := <-watcher.Errors:
				logrus.Errorf("%v", err)

			}
		}
	}()
}

var building bool

func Autobuild(files []string) {
	if building {
		colorPrint("cyan", "still in building...")
		return
	}
	building = true
	defer func() {
		building = false
	}()
	colorPrint("cyan", "Start building...")

	if err := os.Chdir(currpath); err != nil {
		logrus.Errorf("Chdir Error: %+v\n", err)
		return
	}
	cmdName := "go"
	args := []string{"build"}
	args = append(args, "-o", cfg.Output)
	args = append(args, files...)

	cmd := exec.Command(cmdName, args...)
	cmd.Env = append(os.Environ(), "GOGC=off")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logrus.Errorf("%+v\n", err)
		return
	}
	colorPrint(yellow, "Build was successful...")
	Restart(cfg.Output)
}

func Restart(output string) {
	kill()
	go start(output)

}

func start(appname string) {
	colorPrint(red, "Restart...")
	cmd = exec.Command(appname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		_ = cmd.Run()
	}()
	started <- true
}

func kill() {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("Kill.recover -> ", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		// err := cmd.Process.Kill()
		err := killAllProcesses(cmd.Process.Pid)
		if err != nil {
			fmt.Println("Kill -> ", err)
		}
	}
}

func killAllProcesses(pid int) (err error) {
	hasAllKilled := make(chan bool)
	go func() {
		pids, err := psTree(pid)
		if err != nil {
			log.Fatalf("getting all sub processes error: %v\n", err)
			return
		}
		logrus.Debugf("main pid: %d", pid)
		logrus.Debugf("pids: %+v", pids)

		for _, subPid := range pids {
			_ = killProcess(subPid)
		}

		waitForProcess(pid, hasAllKilled)
	}()

	// finally kill the main process
	<-hasAllKilled
	logrus.Debugf("killing MAIN process pid: %d", pid)
	err = cmd.Process.Kill()
	if err != nil {
		return
	}
	logrus.Debugf("kill MAIN process succeed")

	return
}

func killProcess(pid int) (err error) {
	logrus.Debugf("killing process pid: %d", pid)
	pros, err := os.FindProcess(pid)
	if err != nil {
		logrus.Errorf("find process %d error: %v\n", pid, err)
		return
	}
	err = pros.Kill()
	if err != nil {
		logrus.Errorf("killing process %d error: %v\n", pid, err)
		// retry
		time.AfterFunc(2*time.Second, func() {
			logrus.Debugf("retry killing process pid: %d", pid)
			_ = killProcess(pid)
		})
		return
	}
	return
}

func psTree(rootPid int) (res []int, err error) {
	pidOfInterest := map[int]struct{}{rootPid: {}}
	pss, err := ps.Processes()
	if err != nil {
		fmt.Println("ERROR: ", err)
		return
	}

	// we must sort the ps by ppid && pid first, otherwise we probably will miss some sub-processes
	// of the root process during for-range searching
	sort.Slice(pss, func(i, j int) bool {
		ppidLess := pss[i].PPid() < pss[j].PPid()
		pidLess := pss[i].PPid() == pss[j].PPid() && pss[i].Pid() < pss[j].Pid()

		return ppidLess || pidLess
	})

	for _, pros := range pss {
		ppid := pros.PPid()
		if _, exists := pidOfInterest[ppid]; exists {
			pidOfInterest[pros.Pid()] = struct{}{}
		}
	}

	for pid := range pidOfInterest {
		if pid != rootPid {
			res = append(res, pid)
		}
	}

	return
}

func waitForProcess(pid int, hasAllKilled chan bool) {
	pids, _ := psTree(pid)
	if len(pids) == 0 {
		hasAllKilled <- true
		return
	}

	logrus.Infof("still waiting for %d processes %+v to exit", len(pids), pids)
	time.AfterFunc(time.Second, func() {
		waitForProcess(pid, hasAllKilled)
	})
}

func getLastTime(path string) int64 {
	path = strings.ReplaceAll(path, "\\", "/")
	f, err := os.Open(path)
	if err != nil {
		logrus.Errorf("Fail to open file[ %s ]\n", err)
		return time.Now().Unix()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		logrus.Errorf("Fail to get file information[ %s ]\n", err)
		return time.Now().Unix()
	}
	return fi.ModTime().Unix()

}

func InWatchExt(name string) bool {
	for _, s := range cfg.WatchExts {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false

}

func collectFile(currpath string, paths *[]string) {
	//readDir()返回的是指定目录的目录信息的有序序列。
	FileInfos, err := ioutil.ReadDir(currpath)
	if err != nil {
		logrus.Fatal(err)
		return
	}
	useDirectory := false
	for _, FileInfo := range FileInfos {
		if strings.HasSuffix(FileInfo.Name(), "docs") {
			continue
		}
		if strings.HasSuffix(FileInfo.Name(), "swagger") {
			continue
		}
		//是否是排除的路径。
		if isExcluded(path.Join(currpath, FileInfo.Name())) {
			continue
		}
		if FileInfo.IsDir() && FileInfo.Name()[0] != '.' {
			collectFile(currpath+"/"+FileInfo.Name(), paths)
			continue
		}
		if useDirectory {
			continue
		}
		*paths = append(*paths, currpath)
		useDirectory = true
	}
}

func isExcluded(join string) bool {
	absFilePath, err := filepath.Abs(join)
	if err != nil {
		logrus.Errorf("Getting absolute path %s error \n", join)
	}
	for _, p := range cfg.ExcludedPaths {
		absP, err := filepath.Abs(p)
		if err != nil {
			logrus.Fatal(err)
			continue
		}
		if strings.HasPrefix(absFilePath, absP) {
			logrus.Infof("Excluding from watch [%s]", join)
			return true
		}
	}
	return false
}
