//go:build windows

package desktop

func ShowError(title string, err error) {
	if err == nil {
		return
	}
	messageBox(0, title, err.Error(), mbOK|mbIconError)
}
