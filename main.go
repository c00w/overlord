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

func main() {

	log.Printf("starting")
	run("modprobe", "v4l2_common")
	run("ffmpeg")
	http.HandleFunc("/", h)
	log.Fatal(http.ListenAndServe(":80", nil))
}
