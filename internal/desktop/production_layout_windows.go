//go:build windows

package desktop

func productionContentGeometry(clientWidth, clientHeight int) (x, y, width, height int) {
	pad := 16
	x = productionSidebarWidth + pad
	y = productionHeaderHeight
	width = clientWidth - x - pad
	height = clientHeight - productionHeaderHeight - pad
	if width < 760 {
		width = 760
	}
	if height < 620 {
		height = 620
	}
	return
}

func productionWorkColumns(width int) (left, center, right, gap int) {
	gap = 14
	left = width * 21 / 100
	right = width * 23 / 100
	if left < 220 {
		left = 220
	}
	if left > 260 {
		left = 260
	}
	if right < 240 {
		right = 240
	}
	if right > 300 {
		right = 300
	}
	center = width - left - right - gap*2
	if center < 360 {
		shortfall := 360 - center
		shrinkLeft := shortfall / 2
		shrinkRight := shortfall - shrinkLeft
		left -= shrinkLeft
		right -= shrinkRight
		if left < 190 {
			left = 190
		}
		if right < 210 {
			right = 210
		}
		center = width - left - right - gap*2
	}
	return
}

func (s *Shell) layoutProductionWork(x, y, width, height int) {
	left, center, right, gap := productionWorkColumns(width)
	xLeft := x
	xCenter := x + left + gap
	xRight := xCenter + center + gap
	top := y + 10
	bottom := top + height - 20

	// Project rail: repository switching and project metadata stay visible while
	// the central task surface changes.
	moveWindow(s.controls[idProjectsLabel], xLeft+4, top+2, left-8, 20)
	projectListBottom := bottom - 138
	if projectListBottom < top+180 {
		projectListBottom = top + 180
	}
	moveWindow(s.controls[idProjectList], xLeft+4, top+28, left-8, projectListBottom-(top+28))
	moveWindow(s.controls[idAddProject], xLeft+4, projectListBottom+10, left-8, 32)
	nameY := projectListBottom + 50
	moveWindow(s.controls[idProjectName], xLeft+4, nameY, left-92, 30)
	moveWindow(s.controls[idRenameProject], xLeft+left-82, nameY, 78, 30)
	moveWindow(s.controls[idPinProject], xLeft+4, nameY+38, (left-14)/2, 30)
	moveWindow(s.controls[idRemoveProject], xLeft+10+(left-14)/2, nameY+38, (left-14)/2, 30)

	// Primary autonomous-work surface.
	moveWindow(s.controls[idActiveProject], xCenter+4, top+2, center-8, 26)
	moveWindow(s.controls[idActivePath], xCenter+4, top+30, center-8, 18)
	moveWindow(s.controls[idSummary], xCenter+4, top+52, center-8, 20)
	moveWindow(s.controls[idIntentLabel], xCenter+4, top+82, center-8, 20)
	moveWindow(s.controls[idIntent], xCenter+4, top+106, center-8, 66)
	moveWindow(s.controls[idDelegate], xCenter+4, top+180, 196, 34)

	moveWindow(s.controls[idTasksLabel], xCenter+4, top+226, center-8, 20)
	moveWindow(s.controls[idTaskList], xCenter+4, top+250, center-8, 90)
	moveWindow(s.controls[idTaskStatus], xCenter+4, top+348, center-8, 20)
	moveWindow(s.controls[idNextAction], xCenter+4, top+372, center-8, 34)
	moveWindow(s.controls[idReportLabel], xCenter+4, top+414, center-8, 20)

	answerY := bottom - 34
	attentionY := answerY - 22
	buttonY := attentionY - 38
	reportY := top + 438
	reportH := buttonY - 10 - reportY
	if reportH < 72 {
		reportH = 72
	}
	moveWindow(s.controls[idReport], xCenter+4, reportY, center-8, reportH)

	buttonX := xCenter + 4
	availableButtons := center - 8
	cancelW, reviewW, retryW := 78, 102, 102
	copyW := availableButtons - cancelW - reviewW - retryW - 24
	if copyW < 104 {
		copyW = 104
	}
	moveWindow(s.controls[idCancelTask], buttonX, buttonY, cancelW, 30)
	moveWindow(s.controls[idOpenReview], buttonX+cancelW+8, buttonY, reviewW, 30)
	moveWindow(s.controls[idRetryReview], buttonX+cancelW+reviewW+16, buttonY, retryW, 30)
	moveWindow(s.controls[idCopyBranch], buttonX+cancelW+reviewW+retryW+24, buttonY, copyW, 30)
	moveWindow(s.controls[idAttentionLabel], xCenter+4, attentionY, center-8, 18)
	answerW := center - 158
	if answerW < 220 {
		answerW = 220
	}
	moveWindow(s.controls[idAnswer], xCenter+4, answerY, answerW, 30)
	moveWindow(s.controls[idResumeTask], xCenter+center-146, answerY, 142, 30)

	// Durable context remains visible alongside task work rather than hiding in
	// a modal or separate editor.
	moveWindow(s.controls[idNotesLabel], xRight+4, top+2, right-8, 20)
	moveWindow(s.controls[idNotes], xRight+4, top+28, right-8, bottom-top-76)
	moveWindow(s.controls[idSaveNotes], xRight+4, bottom-38, right-8, 32)
}

func (s *Shell) layoutProductionSettings(x, y, width, height int) {
	gap := 16
	left := (width - gap) / 2
	right := width - left - gap
	xRight := x + left + gap
	top := y + 10

	// Left card: worker inventory, usage policy, bridge and routing.
	moveWindow(s.controls[idSettingsTitle], x+4, top, left-8, 1)
	moveWindow(s.controls[idProvidersLabel], x+4, top+2, left-8, 20)
	moveWindow(s.controls[idProviderList], x+4, top+28, left-8, 130)
	moveWindow(s.controls[idConnectProvider], x+4, top+168, 146, 32)
	moveWindow(s.controls[idRescanProviders], x+158, top+168, 94, 32)
	moveWindow(s.controls[idProtectWork], x+4, top+208, left-8, 24)
	moveWindow(s.controls[idAllowMetered], x+4, top+236, left-8, 24)
	moveWindow(s.controls[idMCPLabel], x+4, top+272, left-8, 20)
	moveWindow(s.controls[idMCPStatus], x+4, top+296, left-8, 56)
	moveWindow(s.controls[idCopyMCP], x+4, top+360, 174, 32)
	moveWindow(s.controls[idRunnerLabel], x+4, top+408, left-8, 20)
	moveWindow(s.controls[idRunnerHost], x+4, top+432, left-8, 30)
	moveWindow(s.controls[idHarnessLabel], x+4, top+472, left-8, 20)
	moveWindow(s.controls[idHarnessCommand], x+4, top+496, left-8, 30)
	moveWindow(s.controls[idNotifyLabel], x+4, top+536, left-8, 20)
	moveWindow(s.controls[idNotifyCommand], x+4, top+560, left-8, 30)
	moveWindow(s.controls[idSaveRouting], x+4, top+602, 176, 32)

	// Right card: publication controls, local encrypted secrets and updater.
	moveWindow(s.controls[idReviewLabel], xRight+4, top+2, right-8, 20)
	moveWindow(s.controls[idPublishReviews], xRight+4, top+30, right-8, 24)
	moveWindow(s.controls[idReviewRemote], xRight+4, top+62, right-8, 30)
	moveWindow(s.controls[idSaveReviewPolicy], xRight+4, top+102, 164, 32)
	moveWindow(s.controls[idVaultLabel], xRight+4, top+154, right-8, 20)
	moveWindow(s.controls[idSecretName], xRight+4, top+182, right-8, 30)
	moveWindow(s.controls[idSecretValue], xRight+4, top+220, right-8, 30)
	moveWindow(s.controls[idSaveSecret], xRight+4, top+260, 146, 32)
	secretListH := height - 430
	if secretListH < 120 {
		secretListH = 120
	}
	if secretListH > 178 {
		secretListH = 178
	}
	moveWindow(s.controls[idSecretList], xRight+4, top+302, right-8, secretListH)
	maintenanceY := top + 318 + secretListH
	moveWindow(s.controls[idMaintenanceLabel], xRight+4, maintenanceY, right-8, 20)
	moveWindow(s.controls[idRunUpdater], xRight+4, maintenanceY+28, 226, 34)
}
