package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
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
		"-t", "61",
		"-i", "/dev/video0",
		"-c", "copy",
		"-f", "segment",
		"-segment_time", "60",
		"-strftime", "1", "/data/%s.mkv")
}

func main() {

	log.Printf("starting")
	run("modprobe", "v4l2_common")
	run("df -h")
	go record()
	http.HandleFunc("/", h)
	log.Fatal(http.ListenAndServe(":80", nil))
}
