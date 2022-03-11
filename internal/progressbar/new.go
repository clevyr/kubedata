package progressbar

import (
	"github.com/clevyr/kubedb/internal/terminal"
	"github.com/schollz/progressbar/v3"
	"os"
	"time"
)

func New(max int64) *progressbar.ProgressBar {
	if !terminal.IsTTY() {
		return progressbar.NewOptions64(
			max,
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(10),
			progressbar.OptionThrottle(2*time.Second),
			progressbar.OptionShowCount(),
			progressbar.OptionSpinnerType(14),
		)
	}
	return progressbar.DefaultBytes(max)
}
