package bgptools

import (
	"context"
	"fmt"

	"github.com/Dolyyyy/huma_golang_api_template/internal/modulekit"
)

var (
	globalConfig moduleConfig
	globalErr    error
	globalSvc    *service
)

func init() {
	globalConfig, globalErr = loadConfig()
	if globalErr == nil {
		globalSvc = newService(globalConfig)
		globalSvc.startUpdater()

		if globalConfig.Preload {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), globalConfig.HTTPTimeout)
				defer cancel()
				_ = globalSvc.reload(ctx, true)
			}()
		}
	}

	modulekit.Register(modulekit.Definition{
		ID:          "bgptools",
		Name:        "BGP.tools lookup",
		Description: "Load BGP.tools prefix/ASN datasets and expose IP/ASN lookup endpoints with RAM+disk cache.",
		Validate:    validate,
		HumaRoutes:  registerHumaRoutes,
	})
}

func validate() error {
	if globalErr != nil {
		return globalErr
	}
	if globalSvc == nil {
		return fmt.Errorf("bgptools service is not initialized")
	}

	return nil
}
