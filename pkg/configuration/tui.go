package configuration

var CURRENT_INTRO_VERSION = 1

func TUIShouldShowIntro() bool {
	return configContents.TUI.IntroVersionSeen != CURRENT_INTRO_VERSION
}

func TUIMarkIntroSeen() {
	configContents.TUI.IntroVersionSeen = CURRENT_INTRO_VERSION
	writeConfig(configContents)
}
