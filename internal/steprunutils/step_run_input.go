package steprunutils

import "github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"

func HasNoInput(srd *dbsqlc.GetStepRunDataForEngineRow) bool {
	in := srd.Input
	return len(in) == 0 || string(in) == "{}"

}
