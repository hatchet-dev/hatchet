package olap

import (
	"context"
)

func (oc *OLAPControllerImpl) runAnalyze(ctx context.Context) func() {
	return func() {
		oc.l.Debug().Msgf("analyze: running analyze on partitioned tables")

		err := oc.repo.OLAP().AnalyzeOLAPTables(ctx)

		if err != nil {
			oc.l.Error().Err(err).Msg("could not analyze OLAP tables")
		}
	}
}
