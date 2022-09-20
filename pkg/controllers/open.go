/*
Copyright © 2021 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controllers

import (
	"errors"
	"fmt"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/skratchdot/open-golang/open"
)

func OpenDashboard(options models.ScopedOptions) Error {
	url := options.DashboardHost.Value
	if url == "" {
		// the dashboard host is set during login (though it also has a default)
		utils.HandleError(errors.New("You must login first"))
	}

	project := options.EnclaveProject.Value
	config := options.EnclaveConfig.Value
	if project != "" && config != "" {
		url = url + fmt.Sprintf("/workplace/projects/%s/configs/%s", project, config)
	}
	err := open.Run(url)
	if err != nil {
		return Error{Err: err, Message: "Unable to open dashboard url"}
	}

	return Error{}
}
