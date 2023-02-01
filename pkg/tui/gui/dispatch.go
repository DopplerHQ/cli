package gui

import (
	"context"
	"sort"

	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/tui/gui/state"
	"golang.org/x/sync/errgroup"
)

// Helper functions for fetching -----------------------------------------------
// TODO: Should these keep track of the context for pending requests and cancel previous ones as new ones come in?

func (gui *Gui) fetchConfigs(projectName string) ([]models.ConfigInfo, controllers.Error) {
	fetchOpts := gui.Opts
	fetchOpts.EnclaveProject = models.ScopedOption{Scope: "", Source: "tui", Value: projectName}
	return controllers.GetConfigs(fetchOpts)
}

func (gui *Gui) fetchSecrets(projectName string, configName string) (map[string]models.ComputedSecret, controllers.Error) {
	fetchOpts := gui.Opts
	fetchOpts.EnclaveProject = models.ScopedOption{Scope: "", Source: "tui", Value: projectName}
	fetchOpts.EnclaveConfig = models.ScopedOption{Scope: "", Source: "tui", Value: configName}
	return controllers.GetSecrets(fetchOpts)
}

func (gui *Gui) postSecrets(projectName string, configName string, changeRequests []models.ChangeRequest) (map[string]models.ComputedSecret, controllers.Error) {
	fetchOpts := gui.Opts
	fetchOpts.EnclaveProject = models.ScopedOption{Scope: "", Source: "tui", Value: projectName}
	fetchOpts.EnclaveConfig = models.ScopedOption{Scope: "", Source: "tui", Value: configName}
	return controllers.SetSecrets(fetchOpts, changeRequests)
}

// Helper functions for converting models --------------------------------------

func createSecrets(computedSecrets map[string]models.ComputedSecret) []state.Secret {
	var secrets []state.Secret
	for _, cs := range computedSecrets {
		if cs.Name == "DOPPLER_CONFIG" || cs.Name == "DOPPLER_ENVIRONMENT" || cs.Name == "DOPPLER_PROJECT" {
			continue
		}
		secrets = append(secrets, state.Secret{
			Name:  cs.Name,
			Value: cs.RawValue,
		})
	}

	sort.Sort(state.ByName(secrets))
	return secrets
}

func (gui *Gui) handleError(err error) {
	gui.setIsFetching(false)
	gui.statusMessage = err.Error()
	gui.renderAllStateBasedComponents()
}

// Dispatchable actions --------------------------------------------------------

func (gui *Gui) load() {
	defer recoverScreenOnCrash()
	gui.setIsFetching(true)

	var projectIds []string
	var configInfos []models.ConfigInfo
	var computedSecrets map[string]models.ComputedSecret

	var selectedProjectIdx int
	var selectedConfigIdx int

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		defer recoverScreenOnCrash()
		var err controllers.Error
		projectIds, err = controllers.GetProjectIDs(gui.Opts)
		return err.Unwrap()
	})
	g.Go(func() error {
		defer recoverScreenOnCrash()
		var err controllers.Error
		configInfos, err = gui.fetchConfigs(gui.Opts.EnclaveProject.Value)
		return err.Unwrap()
	})
	g.Go(func() error {
		defer recoverScreenOnCrash()
		var err controllers.Error
		computedSecrets, err = gui.fetchSecrets(gui.Opts.EnclaveProject.Value, gui.Opts.EnclaveConfig.Value)
		return err.Unwrap()
	})
	if err := g.Wait(); err != nil {
		gui.handleError(err)
		return
	}

	projects := make([]state.Project, len(projectIds))
	for idx, id := range projectIds {
		projects[idx] = state.Project{Name: id}
		if id == gui.Opts.EnclaveProject.Value {
			selectedProjectIdx = idx
		}
	}

	configs := make([]state.Config, len(configInfos))
	for idx, configInfo := range configInfos {
		configs[idx] = state.Config{Name: configInfo.Name}
		if configInfo.Name == gui.Opts.EnclaveConfig.Value {
			selectedConfigIdx = idx
		}
	}

	secrets := createSecrets(computedSecrets)

	state.SetProjects(projects)
	state.SetConfigs(configs)
	state.SetSecrets(secrets, gui.Opts.EnclaveProject.Value, gui.Opts.EnclaveConfig.Value)

	gui.cmps.projects.SelectIdx(selectedProjectIdx)
	gui.cmps.configs.SelectIdx(selectedConfigIdx)

	gui.setIsFetching(false)
}

func (gui *Gui) selectProject(projectIdx int) {
	defer recoverScreenOnCrash()
	gui.setIsFetching(true)

	var configInfos []models.ConfigInfo

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		defer recoverScreenOnCrash()
		var err controllers.Error
		configInfos, err = gui.fetchConfigs(state.Projects()[gui.cmps.projects.selectedIdx].Name)
		return err.Unwrap()
	})
	if err := g.Wait(); err != nil {
		gui.handleError(err)
		return
	}

	configs := make([]state.Config, len(configInfos))
	for idx, configInfo := range configInfos {
		configs[idx] = state.Config{Name: configInfo.Name}
	}

	state.SetConfigs(configs)
	state.SetSecrets(make([]state.Secret, 0), "", "")

	gui.setIsFetching(false)
	gui.focusComponent(gui.cmps.configs)
}

func (gui *Gui) selectConfig(configIdx int) {
	defer recoverScreenOnCrash()
	gui.setIsFetching(true)

	var computedSecrets map[string]models.ComputedSecret

	curProj := state.Projects()[gui.cmps.projects.selectedIdx].Name
	curConf := state.Configs()[gui.cmps.configs.selectedIdx].Name

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		defer recoverScreenOnCrash()
		var err controllers.Error
		computedSecrets, err = gui.fetchSecrets(curProj, curConf)
		return err.Unwrap()
	})
	if err := g.Wait(); err != nil {
		gui.handleError(err)
		return
	}

	secrets := createSecrets(computedSecrets)
	state.SetSecrets(secrets, curProj, curConf)

	gui.setIsFetching(false)
	gui.focusComponent(gui.cmps.secrets)
}

func (gui *Gui) saveSecrets(changeRequests []models.ChangeRequest) {
	defer recoverScreenOnCrash()
	gui.setIsFetching(true)

	var computedSecrets map[string]models.ComputedSecret

	curProj := state.Projects()[gui.cmps.projects.selectedIdx].Name
	curConf := state.Configs()[gui.cmps.configs.selectedIdx].Name

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		defer recoverScreenOnCrash()
		var err controllers.Error
		computedSecrets, err = gui.postSecrets(curProj, curConf, changeRequests)
		return err.Unwrap()
	})
	if err := g.Wait(); err != nil {
		gui.handleError(err)
		gui.focusComponent(gui.cmps.secrets)
		return
	}

	secrets := createSecrets(computedSecrets)
	state.SetSecrets(secrets, curProj, curConf)

	gui.setIsFetching(false)
	gui.focusComponent(gui.cmps.secrets)
}
