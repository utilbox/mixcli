package commands

import (
	"errors"
	"fmt"
	"github.com/cheggaaa/pb/v3"
	"github.com/manifoldco/promptui"
	"github.com/mix-go/mixcli/logic"
	"github.com/mix-go/xcli/flag"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	CLI     = "cli"
	API     = "api"
	Web     = "web"
	gRPC    = "grpc"
	None    = "none"
	Gorm    = "gorm"
	Xorm    = "xorm"
	Zap     = "zap"
	Logrus  = "logrus"
	GoRedis = "go-redis"
	Yes     = "yes"
	No      = "no"
)

type NewCommand struct {
}

func (t *NewCommand) Main() {
	name := flag.Arguments().First().String("hello")

	promp := func(label string, items []string) string {
		prompt := promptui.Select{
			Label: label,
			Items: items,
		}
		prompt.HideSelected = true
		_, result, err := prompt.Run()
		if err != nil {
			return ""
		}
		return result
	}

	selectType := CLI
	switch promp("Select project type", []string{"CLI", "API", "Web (contains the websocket)", "gRPC"}) {
	case "CLI":
		selectType = CLI
		break
	case "API":
		selectType = API
		break
	case "Web (contains the websocket)":
		selectType = Web
		break
	case "gRPC":
		selectType = gRPC
		break
	default:
		return
	}

	useDotenv := promp("Use .env configuration file", []string{Yes, No})
	useConf := promp("Use .yml, .json, .toml configuration files", []string{Yes, No})

	var selectLog string
	var selectLogItems []string
	if selectType == CLI {
		selectLogItems = []string{Zap, Logrus, None}
	} else {
		selectLogItems = []string{Zap, Logrus}
	}
	selectLog = promp("Select logger library", selectLogItems)

	var selectDb string
	var selectDbItems []string
	if selectType == API || selectType == Web {
		selectDbItems = []string{Gorm, Xorm}
	} else {
		selectDbItems = []string{Gorm, Xorm, None}
	}
	selectDb = promp("Select database library", selectDbItems)

	var selectRedis string
	selectRedis = promp("Select redis library", []string{GoRedis, None})

	t.NewProject(name, selectType, useDotenv, useConf, selectLog, selectDb, selectRedis)
}

func (t *NewCommand) NewProject(name, selectType, useDotenv, useConf, selectLog, selectDb, selectRedis string) {
	ver := ""
	switch selectType {
	case CLI, API, Web, gRPC:
		ver = fmt.Sprintf("v%s", FrameworkVersion)
		break
	default:
		fmt.Println("Type error, only be console, api, web, grpc")
		return
	}

	echo := "echo $GOPATH"
	if runtime.GOOS == "windows" {
		echo = "echo %PATH%"
	}
	if os.Getenv("GOPATH") == "" {
		fmt.Printf("$GOPATH is not configured, see '%s'\n", echo)
		return
	}

	dr := ":"
	if runtime.GOOS == "windows" {
		dr = ";"
	}
	if strings.Contains(os.Getenv("GOPATH"), dr) {
		fmt.Printf("$GOPATH cannot have multiple directories, see '%s'\n", echo)
		return
	}

	sdir := fmt.Sprintf("%s/pkg/mod/github.com/mix-go/%s-skeleton@%s", os.Getenv("GOPATH"), selectType, ver)
	if _, err := os.Stat(sdir); err != nil {
		cmd := exec.Command("go", "get", fmt.Sprintf("github.com/mix-go/%s-skeleton@%s", selectType, ver))
		fmt.Printf("Skeleton local not found, exec 'go get github.com/mix-go/%s-skeleton@%s'\n", selectType, ver)
		count := 8 * 1024
		current := int64(0)
		bar := pb.StartNew(count)
		go func() {
			path := fmt.Sprintf("%s/pkg/mod/cache/download/github.com/mix-go/%s-skeleton/@v/%s.zip", os.Getenv("GOPATH"), selectType, ver)
			for {
				f, err := os.Open(path)
				if err != nil {
					continue
				}
				fi, err := f.Stat()
				if err != nil {
					_ = f.Close()
					continue
				}
				current = fi.Size()
				bar.SetCurrent(current)
				_ = f.Close()
				time.Sleep(time.Millisecond * 100)
			}
		}()
		err = cmd.Run()
		if err == nil {
			bar.SetTotal(current)
			bar.SetCurrent(current)
		} else {
			bar.SetTotal(0)
			bar.SetCurrent(0)
		}
		bar.Finish()
		if err != nil {
			fmt.Println(fmt.Sprintf("Exec failed: %s", err.Error()))
			fmt.Println("Please try again, or manually execute 'go get ***'")
			return
		}
		time.Sleep(2 * time.Second) // 等待一会，让 gomod 完成解压
		_ = os.Remove(fmt.Sprintf("%s/bin/%s-skeleton", os.Getenv("GOPATH"), selectType))
		fmt.Println(fmt.Sprintf("Skeleton 'github.com/mix-go/%s-skeleton@%s' download is completed", selectType, ver))
	} else {
		fmt.Println(fmt.Sprintf("Skeleton 'github.com/mix-go/%s-skeleton@%s' local found", selectType, ver))
	}

	fmt.Print(" - Generate code")
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dest := fmt.Sprintf("%s/%s", pwd, name)
	if !logic.CopyPath(sdir, dest) {
		panic(errors.New("Copy dir failed"))
	}
	fmt.Println(" > ok")

	if useDotenv == "no" {
		fmt.Print(" - Processing .env")
		if err := logic.ReplaceMain(dest, `_ "github.com/mix-go/cli-skeleton/dotenv"`, ""); err != nil {
			panic(errors.New("Replace failed"))
		}
		_ = os.RemoveAll(fmt.Sprintf("%s/dotenv", dest))
		_ = os.RemoveAll(fmt.Sprintf("%s/.env", dest))
		fmt.Println(" > ok")
	}

	if useConf == "no" {
		fmt.Print(" - Processing conf")
		if err := logic.ReplaceMain(dest, `_ "github.com/mix-go/cli-skeleton/configor"`, ""); err != nil {
			panic(errors.New("Replace failed"))
		}
		_ = os.RemoveAll(fmt.Sprintf("%s/configor", dest))
		_ = os.RemoveAll(fmt.Sprintf("%s/conf", dest))
		fmt.Println(" > ok")
	}

	switch selectLog {
	case Zap:
		if err := logic.ReplaceAll(dest, `logger := di.Logrus`, `logger := di.Zap`); err != nil {
			panic(errors.New("Replace failed"))
		}
		if err := logic.ReplaceAll(dest, `Output: logger.Out`, `Output: &di.ZapOutput{Logger: logger}`); err != nil {
			panic(errors.New("Replace failed"))
		}
		_ = os.Remove(fmt.Sprintf("%s/di/logrus.go", dest))
		break
	case Logrus:
		_ = os.Remove(fmt.Sprintf("%s/di/zap.go", dest))
		break
	case None:
		_ = os.Remove(fmt.Sprintf("%s/di/logrus.go", dest))
		_ = os.Remove(fmt.Sprintf("%s/di/zap.go", dest))
		break
	}

	switch selectDb {
	case Gorm:
		_ = os.Remove(fmt.Sprintf("%s/di/xorm.go", dest))
		break
	case Xorm:
		_ = os.Remove(fmt.Sprintf("%s/di/gorm.go", dest))
		break
	case None:
		_ = os.Remove(fmt.Sprintf("%s/di/gorm.go", dest))
		_ = os.Remove(fmt.Sprintf("%s/di/xorm.go", dest))
		break
	}

	switch selectRedis {
	case GoRedis:
		break
	case None:
		_ = os.Remove(fmt.Sprintf("%s/di/goredis.go", dest))
		break
	}

	fmt.Print(" - Processing package name")
	if err := logic.ReplaceAll(dest, fmt.Sprintf("github.com/mix-go/%s-skeleton", selectType), name); err != nil {
		panic(errors.New("Replace failed"))
	}
	if err := logic.ReplaceMod(dest); err != nil {
		panic(errors.New("Replace go.mod failed"))
	}
	fmt.Println(" > ok")

	fmt.Println(fmt.Sprintf("Project '%s' is generated", name))
}
