package main

import (
	"crypto/rand"
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
	"golang.org/x/crypto/chacha20poly1305"
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
			"-segment_time", "2",
			"-strftime", "1", "/data/raw/%s.mkv")
		c.Stdout = b
		c.Stderr = b
		err := c.Run()
		log.Print("FFMPEG exited?")
		log.Printf("Exit code: %v", err)
		log.Printf("Log buffer: %s", b.String())
		run("df", "-h")
	}
}

func encrypt() {
	for range time.NewTicker(time.Second).C {
		fis, err := ioutil.ReadDir("/data/raw")
		if err != nil {
			log.Fatalf("Error stating /data: %v", err)
			continue
		}
		if len(fis) < 2 {
			continue
		}

		// Make sure we handle the newest entry
		sort.Slice(fis, func(i, j int) bool {
			return fis[i].Name() < fis[j].Name()
		})

		k := make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			log.Fatalf("Unable to read random data: %v", err)
		}

		aed, err := chacha20poly1305.New(k)
		if err != nil {
			log.Fatalf("Unable to create AED: %v", err)
		}
		nonce := make([]byte, aed.NonceSize())
		if _, err := rand.Read(nonce); err != nil {
			log.Fatalf("Unable to read random data: %v", err)
		}

		b, err := ioutil.ReadFile("/data/raw/" + fis[0].Name())
		if err != nil {
			log.Fatalf("Unable to read: %v", err)
		}

		out := aed.Seal(nil, nonce, b, nil)

		if err := ioutil.WriteFile("/data/encrypted/"+fis[0].Name(), out, 0777); err != nil {
			log.Fatalf("Error writing to new file: %v", err)
		}
		if err := os.Remove("/data/raw/" + fis[0].Name()); err != nil {
			log.Fatalf("Error deleting encrypted file")
		}

	}
}

func prune() {
	for range time.NewTicker(time.Second).C {
		fis, err := ioutil.ReadDir("/data/encrypted")
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
				run("df", "-h")
				run("rm", "/data/encrypted/*")
				run("rm", "/data/raw/*")
				log.Fatalf("File /data/encrypted/%q has been around for %v", fi.Name(), o)
			}
		}
		for {
			if size < 200*1024*1024 {
				break
			}
			target := "/data/encrypted/" + fis[0].Name()
			if err := os.Remove(target); err != nil {
				log.Printf("Error removing %s: %v", target, err)
			}
			size -= fis[0].Size()
			fis = fis[1:]
		}
	}
}

func main() {
	log.Printf("starting")
	run("modprobe", "bcm2835-v4l2")
	if err := os.MkdirAll("/data/raw", 0777); err != nil {
		log.Fatalf("Unabel to make /data/raw: %v", err)
	}
	if err := os.MkdirAll("/data/encrypted", 0777); err != nil {
		log.Fatalf("Unabel to make /data/encrypted: %v", err)
	}
	go record()
	go encrypt()
	go prune()
	http.HandleFunc("/", h)
	log.Fatal(http.ListenAndServe(":80", nil))
}
