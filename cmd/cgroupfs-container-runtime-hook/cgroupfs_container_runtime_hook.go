package main

import "C"
import (
	"encoding/json"
	"flag"
	"fmt"
	spec "github.com/opencontainers/runtime-spec/specs-go"
	"io/ioutil"
	"strings"
	//"github.com/zimmski/osutil"

	_ "github.com/openkhal/cgroupfs/pkg/nsenter"

	"k8s.io/klog"
	"log"
	"os"
	"path"
	"runtime"
	"time"
)

///*
//extern void enter_namespace(void);
//*/
//import "C"

var (
	logFileName = "/var/log/cgroupfs-container-runtime-hook.log"
)

func setLog() {
	logfile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(logfile)
	log.SetPrefix(time.Now().Format("2006-01-02 15:04:05") + "[" + fmt.Sprintf("%d", time.Now().UnixNano()) + "]" + " [Prestart] ")
}

var (
	debugflag  = flag.Bool("debug", false, "enable debug output")
	configflag = flag.String("config", "", "configuration file")

	defaultPATH = []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}
)

func exit() {
	if err := recover(); err != nil {
		if _, ok := err.(runtime.Error); ok {
			klog.Errorln(err)
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func doPrestart() {
	defer exit()

	var hookSpec map[string]interface{}
	hookSpecBuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Panicln(err)
		return
	}
	decoder := json.NewDecoder(strings.NewReader(string(hookSpecBuf)))
	decoder.UseNumber()
	decoder.Decode(&hookSpec)

	//pidElem, exists := hookSpec["pid"]
	//if !exists {
	//	log.Printf("No pid exists in hook spec\n")
	//	return
	//}
	//pidJSON, ok := pidElem.(json.Number)
	//if !ok {
	//	log.Printf("No pid exists in hook spec\n")
	//	return
	//}
	//pid, _ := pidJSON.Int64()

	bundleElem, exists := hookSpec["bundle"]
	if !exists {
		log.Panicln("Did not find bundle in hookSpec")
		return
	}
	bundle, ok := bundleElem.(string)
	if !ok {
		log.Panicln("Bundle is not a string")
		return
	}
	specFile := path.Join(bundle, "config.json")
	containerSpec, err := loadSpec(specFile)
	if err != nil {
		klog.Error(err)
		return
	}

	// TODO mount cgroupfs
	//src := ""
	//dst := path.Join(containerSpec.Root.Path, src)
}

func loadSpec(path string) (spec spec.Spec, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&spec); err != nil {
		return
	}
	if spec.Version == "" {
		err = fmt.Errorf("version is empty in OCI spec")
		return
	}
	if spec.Process == nil {
		err = fmt.Errorf("process is empty in OCI spec")
		return
	}
	if spec.Root == nil {
		err = fmt.Errorf("root is empty in OCI spec")
		return
	}

	return
}

//func init() {
//	runtime.GOMAXPROCS(1)
//	runtime.LockOSThread()
//	setLog()
//
//	hookSpecBuf, err := ioutil.ReadAll(os.Stdin)
//	if err != nil {
//		log.Panicln(err)
//		return
//	}
//	decoder := json.NewDecoder(strings.NewReader(string(hookSpecBuf)))
//	decoder.UseNumber()
//	decoder.Decode(&hookSpec)
//	pidElem, exists := hookSpec["pid"]
//	if !exists {
//		log.Printf("No pid exists in hook spec\n")
//		return
//	}
//	pidJSON, ok := pidElem.(json.Number)
//	if !ok {
//		log.Printf("No pid exists in hook spec\n")
//		return
//	}
//	pid, _ = pidJSON.Int64()
//	os.Setenv("mydocker_pid", strconv.FormatInt(pid, 10))
//	//output, _ := osutil.CaptureWithCGo(enterNamespace)
//	//log.Println(string(output))
//}

//func enterNamespace() {
//	C.enter_namespace()
//}

func main() {
	flag.Parse()
	args := flag.Args()
	setLog()

	switch args[0] {
	case "prestart":
		doPrestart()
		os.Exit(0)
	default:
		os.Exit(2)
	}
}
