package model

func (m *AppModel) recalcLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	m.viewport.Width = m.width - 4
	m.textarea.SetWidth(m.width - 6)

	titleHeight := 1
	inputHeight := 7
	statusHeight := 1
	chatHeight := m.height - titleHeight - inputHeight - statusHeight
	if chatHeight < 5 {
		chatHeight = 5
	}
	m.viewport.Height = chatHeight
}
