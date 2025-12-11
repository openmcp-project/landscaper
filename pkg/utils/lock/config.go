package lock

import config "github.com/openmcp-project/landscaper/apis/config/v1alpha1"

func IsLockingEnabledForMainControllers(config *config.LandscaperConfiguration) bool {
	return config != nil &&
		config.HPAMainConfiguration != nil &&
		config.HPAMainConfiguration.MaxReplicas > 1
}
