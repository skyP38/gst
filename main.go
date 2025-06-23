package main

///dev/video2

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-gst/go-gst/gst"
)

func buildPipeline() (*gst.Pipeline, error) {
	pipeline, err := gst.NewPipeline("")
	if err != nil {
		return nil, err
	}

	elements := make([]*gst.Element, 0)

	v4l2src, err := gst.NewElement("v4l2src")
	if err != nil {
		return nil, err
	}
	v4l2src.SetProperty("device", "/dev/video2")
	elements = append(elements, v4l2src)

	caps, err := gst.NewElement("capsfilter")
	if err != nil {
		return nil, err
	}
	caps.SetProperty("caps", gst.NewCapsFromString(
		"image/jpeg,width=1920,height=1080,framerate=30/1",
	))
	elements = append(elements, caps)

	queue, err := gst.NewElement("queue")
	if err != nil {
		return nil, err
	}
	queue.SetProperty("max-size-buffers", uint(3))
	queue.SetProperty("leaky", "downstream")
	elements = append(elements, queue)

	jpegparse, err := gst.NewElement("jpegparse")
	if err != nil {
		return nil, err
	}
	elements = append(elements, jpegparse)

	vaapijpegdec, err := gst.NewElement("vaapijpegdec")
	if err != nil {
		return nil, err
	}
	elements = append(elements, vaapijpegdec)

	vaapipostproc, err := gst.NewElement("vaapipostproc")
	if err != nil {
		return nil, err
	}
	elements = append(elements, vaapipostproc)

	fpsdisplaysink, err := gst.NewElement("fpsdisplaysink")
	if err != nil {
		return nil, err
	}
	fpsdisplaysink.SetProperty("text-overlay", true)
	fpsdisplaysink.SetProperty("sync", false)

	vaapisink, err := gst.NewElement("vaapisink")
	if err != nil {
		return nil, err
	}
	vaapisink.SetProperty("sync", false)
	vaapisink.SetProperty("force-aspect-ratio", true)
	fpsdisplaysink.SetProperty("video-sink", vaapisink)

	elements = append(elements, fpsdisplaysink)

	for _, elem := range elements {
		pipeline.Add(elem)
	}

	for i := 0; i < len(elements)-1; i++ {
		gst.ElementLinkMany(elements[i], elements[i+1])
	}

	return pipeline, nil
}

func main() {
	gst.Init(nil)

	pipeline, err := buildPipeline()
	if err != nil {
		log.Fatal("error build pipeline", err)
	}

	pipeline.SetState(gst.StatePlaying)

	pauseChan := make(chan struct{}, 1)
	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

	go func() {
		fd := int(os.Stdin.Fd())
		termios := enableRawMode(fd)
		defer restoreMode(fd, termios)

		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				continue
			}

			switch buf[0] {
			case ' ':
				select {
				case pauseChan <- struct{}{}:
				default:
				}
			case 3: // Ctrl+C
				quitChan <- syscall.SIGINT
			}
		}
	}()

	bus := pipeline.GetPipelineBus()

	for {
		select {
		case <-pauseChan:
			handlePause(pipeline)
		case <-quitChan:
			pipeline.SetState(gst.StateNull)
			log.Println("end")
			return
		default:
			handleMessages(bus, pipeline)
		}
	}
}

func handlePause(pipeline *gst.Pipeline) {
	_, state := pipeline.GetState(gst.StatePlaying, gst.ClockTimeNone)
	if state == gst.StatePlaying {
		pipeline.SetState(gst.StatePaused)
		log.Println("PAUSE (enter shift to continue)")
	} else {
		pipeline.SetState(gst.StatePlaying)
		log.Println("PLAY (enter shift to pause)")
	}
}

func handleMessages(bus *gst.Bus, pipeline *gst.Pipeline) {
	for {
		msg := bus.Pop()
		if msg == nil {
			break
		}
		switch msg.Type() {
		case gst.MessageEOS:
			pipeline.SetState(gst.StateNull)
			os.Exit(0)
		case gst.MessageError:
			err := msg.ParseError()
			log.Println("error", err)
			pipeline.SetState(gst.StateNull)
			os.Exit(1)
		}
	}
}
