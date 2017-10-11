package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/armon/circbuf"
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
	log.Print("Running FFMPEG")
	for {
		b, _ := circbuf.NewBuffer(10 * 1024 * 1024)
		c := exec.Command("ffmpeg",
			"-f", "v4l2",
			"-input_format", "h264",
			"-framerate", "30",
			"-video_size", "1920x1080",
			"-i", "/dev/video0",
			"-c", "copy",
			"-f", "segment",
			"-segment_time", "60",
			"-strftime", "1", "/data/%s.mkv")
		c.Stdout = b
		c.Stderr = b
		err := c.Run()
		log.Print("FFMPEG exited?")
		log.Printf("Exit code: %v", err)
		log.Printf("Log buffer: %s", b.String())
	}
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
			n := strings.TrimSuffix(fi.Name(), ".mkv")
			i, err := strconv.ParseInt(n, 10, 64)
			if err != nil {
				log.Printf("Unable to extract timestamp from %q", fi.Name())
				continue
			}
			if o := time.Since(time.Unix(i, 0)); o > 15*time.Minute {
				log.Printf("File %q has been around for %v", fi.Name(), o)
			}
		}
		if size > 800*1024*1024 {
			target := "/data/" + fis[0].Name()
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
