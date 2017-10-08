package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"time"
)

func h(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello World!")
	log.Print(r)
}

func run(arg ...string) {
	out, err := exec.Command(arg[0], arg[1:]...).CombinedOutput()
	log.Printf("Ran %v, err = %v", arg, err)
	log.Printf("Output ----\n %s", string(out))
}

func record() {
	run("ffmpeg",
		"-f", "v4l2",
		"-input_format", "h264",
		"-framerate", "30",
		"-video_size", "1920x1080",
		"-i", "/dev/video0",
		"-c", "copy",
		"-f", "segment",
		"-segment_time", "60",
		"-strftime", "1", "/data/%s.mkv")
}

func prune() {
	for range time.NewTicker(30 * time.Second).C {
		fis, err := ioutil.ReadDir("/data")
		if err != nil {
			log.Printf("Error stating /data: %v", err)
			continue
		}

		// Make sure we delete newest entry
		sort.Slice(fis, func(i, j int) bool {
			return fis[i].Name() < fis[j].Name()
		})

		size := int64(0)
		for _, fi := range fis {
			size += fi.Size()
		}
		log.Printf("Total /data size is %d, %d files present", size, len(fis))
		if size > 800*1024*1024 {
			target := "/data/" + fis[0].Name()
			log.Printf("Pruning %q", target)
			if err := os.Remove(target); err != nil {
				log.Printf("Error removing %s: %v", target, err)
			}
		}
	}
}

func main() {
	log.Printf("starting")
	run("modprobe", "bcm2835-v4l2")
	go record()
	go prune()
	http.HandleFunc("/", h)
	log.Fatal(http.ListenAndServe(":80", nil))
}
