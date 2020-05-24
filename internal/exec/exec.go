package exec

import (
	"os"
	"os/exec"
)

func wrapTelegramNotify(name string, args ...string) (string, []string) {
	_, f1 := os.LookupEnv("TELEGRAM_NOTIFY_TOKEN")
	_, f2 := os.LookupEnv("TELEGRAM_NOTIFY_CHAT_ID")
	if f1 && f2 {
		// todo: make the telegram-notify arguments configurable
		args = append([]string{
			"-id=github-webhook",
			"-success",
			"--",
			name,
		}, args...)
		name = "telegram-notify"
	}
	return name, args
}

func WebsiteBuilder(inputDir string, outputDir string, symlink string) error {
	name, args := wrapTelegramNotify("website-builder", inputDir, outputDir, symlink)
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
