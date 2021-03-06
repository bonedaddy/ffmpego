package ffmpego

import (
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

// A VideoReader decodes a video file using ffmpeg.
type VideoReader struct {
	command *exec.Cmd
	inPipe  *os.File
	info    *VideoInfo
}

func NewVideoReader(path string) (*VideoReader, error) {
	vr, err := newVideoReader(path)
	if err != nil {
		err = errors.Wrap(err, "read video")
	}
	return vr, err
}

func newVideoReader(path string) (*VideoReader, error) {
	info, err := GetVideoInfo(path)
	if err != nil {
		return nil, err
	}

	inPipe, childPipe, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(
		"ffmpeg",
		"-i", path,
		"-f", "rawvideo", "-pix_fmt", "rgb24",
		"pipe:3",
	)
	cmd.ExtraFiles = []*os.File{childPipe}
	if err := cmd.Start(); err != nil {
		inPipe.Close()
		childPipe.Close()
	}
	childPipe.Close()
	return &VideoReader{
		command: cmd,
		inPipe:  inPipe,
		info:    info,
	}, nil
}

// VideoInfo gets information about the current video.
func (v *VideoReader) VideoInfo() *VideoInfo {
	return v.info
}

// ReadFrame reads the next frame from the video.
//
// If the video is finished decoding, nil will be returned
// along with io.EOF.
func (v *VideoReader) ReadFrame() (image.Image, error) {
	buf := make([]byte, 3*v.info.Width*v.info.Height)
	if _, err := io.ReadFull(v.inPipe, buf); err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, v.info.Width, v.info.Height))
	for y := 0; y < v.info.Height; y++ {
		for x := 0; x < v.info.Width; x++ {
			rgb := buf[:3]
			buf = buf[3:]
			img.Set(x, y, &color.RGBA{
				R: rgb[0],
				G: rgb[1],
				B: rgb[2],
				A: 0xff,
			})
		}
	}
	return img, nil
}

// Close stops the decoding process and closes all
// associated files.
func (v *VideoReader) Close() error {
	// When we close the pipe, the subprocess should terminate
	// (possibly with an error) because it cannot write.
	v.inPipe.Close()
	v.command.Wait()
	return nil
}
