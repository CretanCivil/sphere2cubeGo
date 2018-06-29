package main

import (
	"flag"
	"log"
	"os"
	"./cache"
	"./saver"
	"./worker"
	"time"
	"os/exec"
	"bytes"
	"syscall"
	"github.com/jimuyida/glog"
	"fmt"
	"strings"
	"path/filepath"
)

var (
	tileNames = []string{
		worker.TileUp,
		worker.TileDown,
		worker.TileFront,
		worker.TileRight,
		worker.TileBack,
		worker.TileLeft,
	}

	tileSize          = 4096
	originalImagePath = ""
	outPutDir         = "./build"

	tileSizeCmd          = flag.Int("s", tileSize, "Size in px of final tile")
	originalImagePathCmd = flag.String("i", "", "Path to input equirectangular panorama")
	outPutDirCmd         = flag.String("o", outPutDir, "Path to output directory")

	krpanoDir = "C:/Users/WXH/Downloads/krpa_32449/krpa/krpano-1.19-pr13/"
	krpanoExe = krpanoDir+"krpanotools64.exe"
	krpanoArgs = "-config="+krpanoDir+"templates/convertdroplets.config"
	useKrPano = true
)

func main() {

	flag.Parse()
	tileSize = *tileSizeCmd
	originalImagePath = *originalImagePathCmd
	outPutDir = *outPutDirCmd

	if originalImagePath == "" {
		flag.PrintDefaults()
		os.Exit(2)
	}

	_, err := os.Stat(originalImagePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalf("%v not found", originalImagePath)
			os.Exit(2)
		}
	}

	timeStart := time.Now()

	if useKrPano {
		tilesDir := filepath.Dir(originalImagePath)
		filename := filepath.Base(originalImagePath)
		ext := filepath.Ext(originalImagePath)


		shortName := filename[0:len(filename)-len(ext)]
		fmt.Println(tilesDir,filename,ext,shortName)

		tiles := map[string]string{
			"u" : "0",
			"l" : "4",
			"f" : "1",
			"r": "2",
			"b": "3",
			"d": "5",
		}


		err,b,c := cutByKrpano(originalImagePath)
		fmt.Println(err,b,c)
		if err != nil {
			return
		}

		for key, _ := range tiles {
			tileFile := filepath.Join(tilesDir,shortName+"_"+key+".jpg")
			fmt.Println(tileFile)
			finfo, err := os.Stat(tileFile)
			if err != nil || finfo.Size() <= 0{
				return
			}
		}

		for key, value := range tiles {
			tileFile := filepath.Join(tilesDir, shortName+"_"+key+".jpg")
			err = saver.SaveTileFile(tileFile,value, outPutDir)
			log.Printf("Process for tile %v --> finished", value)
		}

	} else {
		done := make(chan worker.TileResult)

		cacheResult := cache.CacheAnglesHandler(tileSize)
		for _, tileName := range tileNames {
			tile := worker.Tile{TileName: tileName, TileSize: tileSize}
			go worker.Worker(tile, cacheResult, originalImagePath, done)
		}

		for range tileNames {
			tileResult := <-done
			err = saver.SaveTile(tileResult, outPutDir)

			if err != nil {
				log.Fatal(err.Error())
				os.Exit(2)
			}

			log.Printf("Process for tile %v --> finished", tileResult.Tile.TileName)
		}
	}


	timeFinish := time.Now()
	duration := timeFinish.Sub(timeStart)
	log.Printf("Time to render: %v seconds", duration.Seconds())
}

func cmdRunWithTimeout(cmd *exec.Cmd, timeout time.Duration) (error, bool) {
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	select {
	case <-time.After(timeout):
		// timeout
		if err = cmd.Process.Kill(); err != nil {
			glog.Error("failed to kill: %s, error: %s", cmd.Path, err)
		}
		go func() {
			<-done // allow goroutine to exit
		}()
		glog.Info("process:%s killed", cmd.Path)
		return err, true
	case err = <-done:
		return err, false
	}
}

func cutByKrpano(filepath string) (err error, exitCode int, outStr string) {
	filepath = strings.Replace(filepath,"\\","/",-1)
//	fmt.Println(krpanoExe,"sphere2cube "+filepath + krpanoArgs)


	cmd := exec.Command(krpanoExe, "sphere2cube",filepath,"-q",  krpanoArgs)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		return err,-1,err.Error()
	}

	var isTimeout bool
	err,isTimeout = cmdRunWithTimeout(cmd,time.Duration(5*time.Minute))
	if isTimeout {
		exitCode = -1
		return nil, exitCode, "time out"
	}
	if err != nil {
		//glog.Error(err)
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			//log.Printf("Could not get exit code for failed program: %v, %v", name, args)
			exitCode = -1
		}
		outStr = out.String()
		return err, exitCode, outStr
	} else {
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	outStr = out.String()
	return nil, exitCode, outStr
}