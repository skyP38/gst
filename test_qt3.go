package main

/*
#cgo pkg-config: gstreamer-1.0 gstreamer-video-1.0

#include <gst/gst.h>
#include <gst/video/videooverlay.h>
#include <gst/video/video.h>

static void set_window_handle(GstElement *sink, guintptr handle) {
    gst_video_overlay_set_window_handle(GST_VIDEO_OVERLAY(sink), handle);
}

static gint64 query_position(GstElement *pipeline) {
    gint64 position = 0;
    gst_element_query_position(pipeline, GST_FORMAT_TIME, &position);
    return position;
}

static gint64 query_duration(GstElement *pipeline) {
    gint64 duration = 0;
    gst_element_query_duration(pipeline, GST_FORMAT_TIME, &duration);
    return duration;
}

static gboolean seek(GstElement *pipeline, gint64 position) {
    return gst_element_seek_simple(pipeline, GST_FORMAT_TIME,
        GST_SEEK_FLAG_FLUSH | GST_SEEK_FLAG_KEY_UNIT, position);
}

static void set_render_rectangle(GstElement *sink, gint x, gint y, gint width, gint height) {
    gst_video_overlay_set_render_rectangle(GST_VIDEO_OVERLAY(sink), x, y, width, height);
}
*/
import "C"
import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

type VideoPlayer struct {
	window      *widgets.QMainWindow
	videoWidget *widgets.QWidget
	playButton  *widgets.QPushButton
	stopButton  *widgets.QPushButton
	pauseButton *widgets.QPushButton
	openButton  *widgets.QPushButton
	winId       uintptr
	pipeline    unsafe.Pointer
	isPlaying   bool
	isPaused    bool
	position    int64
	currentFile string
	fileLabel   *widgets.QLabel
	statusLabel *widgets.QLabel
}

func main() {
	fmt.Println("Запуск приложения...")

	// инициализация Qt
	app := widgets.NewQApplication(len(os.Args), os.Args)

	// инициализация GStreamer
	C.gst_init(nil, nil)

	player := NewVideoPlayer()

	player.window.Show()

	app.Exec()

	fmt.Println("Приложение завершено")
}

func NewVideoPlayer() *VideoPlayer {
	// главное окно
	window := widgets.NewQMainWindow(nil, 0)
	window.SetWindowTitle("Video Player with Scaling")
	window.SetMinimumSize2(800, 600)

	centralWidget := widgets.NewQWidget(nil, 0)
	window.SetCentralWidget(centralWidget)

	// виджет для видео
	videoWidget := widgets.NewQWidget(nil, 0)
	videoWidget.SetStyleSheet("background-color: black;")

	// кнопки
	playButton := widgets.NewQPushButton2("Play", nil)
	stopButton := widgets.NewQPushButton2("Stop", nil)
	pauseButton := widgets.NewQPushButton2("Pause", nil)
	openButton := widgets.NewQPushButton2("Open Video", nil)

	fileLabel := widgets.NewQLabel2("No file selected", nil, 0)
	fileLabel.SetAlignment(core.Qt__AlignCenter)

	statusLabel := widgets.NewQLabel2("Status: Ready", nil, 0)
	statusLabel.SetAlignment(core.Qt__AlignCenter)

	player := &VideoPlayer{
		window:      window,
		videoWidget: videoWidget,
		playButton:  playButton,
		stopButton:  stopButton,
		pauseButton: pauseButton,
		openButton:  openButton,
		fileLabel:   fileLabel,
		statusLabel: statusLabel,
		currentFile: "",
	}

	player.setupUI()

	player.setupConnections()

	return player
}

func (vp *VideoPlayer) setupUI() {
	mainLayout := widgets.NewQVBoxLayout()

	fileInfoLayout := widgets.NewQHBoxLayout()
	fileInfoLayout.AddWidget(widgets.NewQLabel2("File:", nil, 0), 0, 0)
	fileInfoLayout.AddWidget(vp.fileLabel, 1, 0)

	statusLayout := widgets.NewQHBoxLayout()
	statusLayout.AddWidget(widgets.NewQLabel2("Status:", nil, 0), 0, 0)
	statusLayout.AddWidget(vp.statusLabel, 1, 0)

	buttonLayout := widgets.NewQHBoxLayout()
	buttonLayout.AddWidget(vp.openButton, 0, 0)
	buttonLayout.AddWidget(vp.playButton, 0, 0)
	buttonLayout.AddWidget(vp.pauseButton, 0, 0)
	buttonLayout.AddWidget(vp.stopButton, 0, 0)

	mainLayout.AddLayout(fileInfoLayout, 0)
	mainLayout.AddLayout(statusLayout, 0)
	mainLayout.AddWidget(vp.videoWidget, 1, 0)
	mainLayout.AddLayout(buttonLayout, 0)

	vp.window.CentralWidget().SetLayout(mainLayout)
}

func (vp *VideoPlayer) setupConnections() {
	vp.playButton.ConnectClicked(vp.handlePlay)
	vp.stopButton.ConnectClicked(vp.handleStop)
	vp.pauseButton.ConnectClicked(vp.handlePause)
	vp.openButton.ConnectClicked(vp.handleOpen)

	vp.videoWidget.ConnectEvent(vp.handleEvent)
}

// Обработчики событий
func (vp *VideoPlayer) handlePlay(checked bool) {
	if vp.winId == 0 {
		log.Println("Video widget not realized yet")
		vp.statusLabel.SetText("Status: Video widget not ready")
		return
	}

	if vp.currentFile == "" {
		log.Println("No video file selected")
		vp.statusLabel.SetText("Status: No file selected")
		widgets.QMessageBox_Information(vp.window, "Information",
			"Please select a video file first",
			widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
		return
	}

	if vp.isPlaying && !vp.isPaused {
		log.Println("Already playing")
		vp.statusLabel.SetText("Status: Already playing")
		return
	}

	if vp.pipeline != nil && vp.isPaused {
		// продолжение воспроизведение с паузы
		pipeline := (*C.GstElement)(vp.pipeline)
		if C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_SUCCESS {
			vp.isPaused = false
			vp.updatePlaybackState(true)
			log.Println("Playback resumed")
		}
		return
	}

	vp.startPlayback()
}

func (vp *VideoPlayer) handleStop(checked bool) {
	if vp.pipeline != nil {
		pipeline := (*C.GstElement)(vp.pipeline)

		if vp.isPlaying {
			position := C.query_position(pipeline)
			vp.position = int64(position)
			log.Printf("Saving position: %d ns", vp.position)
		}

		C.gst_element_set_state(pipeline, C.GST_STATE_NULL)
		C.gst_object_unref(C.gpointer(pipeline))
		vp.pipeline = nil
		vp.isPlaying = false
		vp.isPaused = false
		vp.updatePlaybackState(false)
		log.Println("Playback stopped")
	}
}

func (vp *VideoPlayer) handlePause(checked bool) {
	if vp.pipeline == nil || !vp.isPlaying {
		return
	}
	pipeline := (*C.GstElement)(vp.pipeline)

	if !vp.isPaused {
		// пауза
		position := C.query_position(pipeline)
		vp.position = int64(position)
		C.gst_element_set_state(pipeline, C.GST_STATE_PAUSED)
		vp.isPaused = true
		vp.updatePlaybackState(false)
		log.Printf("Playback paused at position: %d ns", vp.position)
	} else {
		// продолжение воспроизведение
		if C.gst_element_set_state(pipeline, C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_SUCCESS {
			vp.isPaused = false
			vp.updatePlaybackState(true)
			log.Println("Playback resumed")
		}
	}
}

func (vp *VideoPlayer) handleOpen(checked bool) {
	filePath := widgets.QFileDialog_GetOpenFileName(nil,
		"Select Video File",
		"",
		"Video Files (*.mp4 *.avi *.mkv *.mov);;All Files (*)",
		"", 0)

	if filePath != "" {
		vp.loadVideoFile(filePath)
	}
}

func (vp *VideoPlayer) handleEvent(event *core.QEvent) bool {
	if event.Type() == core.QEvent__Show {
		vp.winId = uintptr(vp.videoWidget.WinId())
		fmt.Printf("Video widget realized - WinId: %d\n", vp.winId)
	}
	return vp.videoWidget.EventDefault(event)
}

// методы для обновления состояния UI
func (vp *VideoPlayer) updatePlaybackState(playing bool) {
	if playing {
		vp.playButton.SetText("Playing")
		vp.statusLabel.SetText("Status: Playing")
	} else {
		vp.playButton.SetText("Play")
		vp.statusLabel.SetText("Status: Paused")
	}
}

func (vp *VideoPlayer) loadVideoFile(filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("Video file not found: %s", filePath)
		vp.statusLabel.SetText("Status: File not found")
		widgets.QMessageBox_Critical(vp.window, "Error",
			fmt.Sprintf("File not found: %s", filePath),
			widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
		return
	}

	vp.handleStop(false)

	vp.currentFile = filePath

	fileName := filepath.Base(filePath)
	vp.fileLabel.SetText(fileName)
	vp.fileLabel.SetToolTip(filePath)
	vp.statusLabel.SetText("Status: File loaded - " + fileName)

	log.Printf("Video file selected: %s", filePath)
}

func (vp *VideoPlayer) startPlayback() {
	vp.handleStop(false)

	pipelineStr := C.CString(fmt.Sprintf(
		"filesrc location=%s ! decodebin ! videoscale ! videoconvert ! video/x-raw,width=%d,height=%d ! ximagesink name=mysink",
		vp.currentFile, vp.videoWidget.Width(), vp.videoWidget.Height()))
	defer C.free(unsafe.Pointer(pipelineStr))

	pipeline := C.gst_parse_launch(pipelineStr, nil)
	if pipeline == nil {
		log.Println("Failed to create pipeline")
		vp.statusLabel.SetText("Status: Failed to create pipeline")
		return
	}
	vp.pipeline = unsafe.Pointer(pipeline)

	sinkName := C.CString("mysink")
	defer C.free(unsafe.Pointer(sinkName))

	sink := C.gst_bin_get_by_name((*C.GstBin)(vp.pipeline), sinkName)
	if sink == nil {
		log.Println("Failed to get sink element")
		vp.statusLabel.SetText("Status: Failed to get sink element")
		C.gst_object_unref(C.gpointer(vp.pipeline))
		vp.pipeline = nil
		return
	}
	defer C.gst_object_unref(C.gpointer(sink))

	C.set_window_handle(sink, C.guintptr(vp.winId))

	// начальный размер области
	size := vp.videoWidget.Size()
	C.set_render_rectangle(sink, C.gint(0), C.gint(0), C.gint(size.Width()), C.gint(size.Height()))

	// запуск pipeline
	if C.gst_element_set_state((*C.GstElement)(vp.pipeline), C.GST_STATE_PLAYING) == C.GST_STATE_CHANGE_FAILURE {
		log.Println("Failed to start pipeline")
		vp.statusLabel.SetText("Status: Failed to start playback")
		C.gst_object_unref(C.gpointer(vp.pipeline))
		vp.pipeline = nil
		return
	}

	vp.isPlaying = true
	vp.isPaused = false
	vp.updatePlaybackState(true)
	log.Println("Playback started with GStreamer API")
}
