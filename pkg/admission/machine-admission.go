package admission

import (
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"fmt"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider"
	"k8s.io/client-go/kubernetes"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
)

type MachineValidationResult struct {
	Error error
	ExitCode int
}

const (
	OK    = iota
	DefaultingError
	ProviderError
	ValidatingError

)

// Defaults and validates machine according to its respective Provider implementation.
//
// Returns a MachineValidationResult with the repsective ExitCode
func ValidateMachine(machine v1alpha1.Machine, kubeClient *kubernetes.Clientset) *MachineValidationResult {
	mvr := MachineValidationResult{}
	prov, err := getProvider(machine, kubeClient)
	if err != nil {
		mvr.Error = err; mvr.ExitCode = ProviderError
		return &mvr
	}
	defaultedSpec, _, err  := prov.AddDefaults(machine.Spec)
	if err != nil {
		mvr.Error =  err; mvr.ExitCode = DefaultingError
		return &mvr
	}
	if err = prov.Validate(defaultedSpec); err != nil {
		mvr.Error = err; mvr.ExitCode = ValidatingError
		return &mvr
	}

	mvr.ExitCode = OK
	return &mvr
}

func getProvider(machine v1alpha1.Machine, kubeClient *kubernetes.Clientset) (cloud.Provider, error) {
	providerConfig, err := providerconfig.GetConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider config: %v", err)
	}
	skg := providerconfig.NewConfigVarResolver(kubeClient)
	prov, err := cloudprovider.ForProvider(providerConfig.CloudProvider, skg)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud provider %q: %v", providerConfig.CloudProvider, err)
	}

	return prov, nil
}