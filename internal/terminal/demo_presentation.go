package terminal

import "fmt"

type demoFrame struct {
	width, height, x, y int
	rows                []string
	status              string
}

func (r *TerminalRenderer) StartLatestPresentation() *LatestFrame[demoFrame] {
	r.latestPresentation = true
	writer := r.Writer
	var previous demoFrame
	return NewLatestFrame(func(frame demoFrame) {
		if frame.width != previous.width || frame.height != previous.height || frame.x != previous.x || frame.y != previous.y {
			fmt.Fprint(writer, "\033[2J")
			previous = demoFrame{}
		}
		for y, row := range frame.rows {
			if y >= len(previous.rows) || row != previous.rows[y] {
				fmt.Fprintf(writer, "\033[%d;%dH%s\033[K", frame.y+y+1, frame.x+1, row)
			}
		}
		if frame.status != previous.status {
			fmt.Fprintf(writer, "\033[%d;%dH\033[K%s", frame.y+frame.height+1, frame.x+1, frame.status)
		}
		if writer.Flush() == nil {
			previous = frame
		}
	})
}

func (r *TerminalRenderer) QueuePresentation(p *LatestFrame[demoFrame]) {
	status := r.statusLine()
	if len(status) > r.Width {
		status = status[:r.Width]
	}
	color := "none"
	if r.UseColor {
		color = r.ColorMode
		if color == "" {
			color = "truecolor"
		}
	}
	rows := make([]string, r.Height)
	for y := range rows {
		rows[y] = EncodeRow(r.Surface[y], r.ColorBuffer[y], color, -1)
	}
	p.Submit(demoFrame{r.Width, r.Height, r.OffsetX, r.OffsetY, rows, status})
}
