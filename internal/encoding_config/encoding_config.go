package encodingconfig

import (
	"cosmossdk.io/simapp/params"
	cudosapp "github.com/CudoVentures/cudos-node/app"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func MakeEncodingConfig() params.EncodingConfig {
	encodingConfig := params.MakeTestEncodingConfig()
	std.RegisterLegacyAminoCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	manager := mergeBasicManagers([]module.BasicManager{cudosapp.ModuleBasics})
	manager.RegisterLegacyAminoCodec(encodingConfig.Amino)
	manager.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}

// mergeBasicManagers merges the given managers into a single module.BasicManager
func mergeBasicManagers(managers []module.BasicManager) module.BasicManager {
	var union = module.BasicManager{}
	for _, manager := range managers {
		for k, v := range manager {
			union[k] = v
		}
	}
	return union
}
