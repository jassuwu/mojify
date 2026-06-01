package media

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"github.com/jass/mojify/packages/core/internal/render"
)

func DecodeArgs(path string, width int, height int) []string {
	widthText := strconv.Itoa(width)
	heightText := strconv.Itoa(height)
	return []string{
		"-v", "error",
		"-i", path,
		"-vf", "scale=" + widthText + ":" + heightText + ":force_original_aspect_ratio=decrease,pad=" + widthText + ":" + heightText + ":(ow-iw)/2:(oh-ih)/2",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
}

func StartDecoder(path string, width int, height int) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command("ffmpeg", DecodeArgs(path, width, height)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdout, nil
}

func ReadRawFrame(r io.Reader, width int, height int) (render.RGBFrame, error) {
	size, err := frameByteSize(width, height)
	if err != nil {
		return render.RGBFrame{}, err
	}
	buf := make([]byte, size)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return render.RGBFrame{}, err
	}
	return render.NewRGBFrame(width, height, buf), nil
}

func frameByteSize(width int, height int) (int, error) {
	if width <= 0 || height <= 0 {
		return 0, fmt.Errorf("invalid frame dimensions %dx%d", width, height)
	}
	if width > int(^uint(0)>>1)/height/3 {
		return 0, fmt.Errorf("frame dimensions overflow %dx%d", width, height)
	}
	return width * height * 3, nil
}
