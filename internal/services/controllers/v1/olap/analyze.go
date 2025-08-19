package olap

import (
	"context"
)

func (oc *OLAPControllerImpl) runAnalyze(ctx context.Context) func() {
	return func() {
		oc.l.Debug().Msgf("analyze: running analyze on partitioned tables")

		tenant, err := oc.p.GetInternalTenantForController(ctx)

		if err != nil {
			oc.l.Error().Err(err).Msg("could not get internal tenant")
			return
		}

		if tenant == nil {
			return
		}

		err = oc.repo.OLAP().AnalyzeOLAPTables(ctx)

		if err != nil {
			oc.l.Error().Err(err).Msg("could not analyze OLAP tables")
		}
	}
}
