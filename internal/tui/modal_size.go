package tui

func modalScreenSize(a *App) (int, int) {
	const (
		defaultWidth  = 80
		defaultHeight = 24
	)
	if a == nil || a.pages == nil {
		return defaultWidth, defaultHeight
	}
	_, _, w, h := a.pages.GetRect()
	if w <= 0 {
		w = defaultWidth
	}
	if h <= 0 {
		h = defaultHeight
	}
	return w, h
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
