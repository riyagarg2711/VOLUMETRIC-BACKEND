package app

import (
	"database/sql"
	"net/http"

	authh "volumetric-backend/internal/auth/handler"
	authmw "volumetric-backend/internal/auth/middleware"
	authr "volumetric-backend/internal/auth/repo"
	"volumetric-backend/internal/handler"
	"volumetric-backend/internal/repo"
)

type App struct {
	AuthHandler    *authh.AuthHandler
	ScanHandler    *handler.ScanHandler
	CoordHandler   *handler.CoordinateHandler
	VolumeHandler  *handler.VolumeHandler
	AuthMiddleware func(http.Handler) http.Handler
}

func New(db *sql.DB) *App {
	authRepo  := authr.NewAuthRepo(db)
	scanRepo  := repo.NewScanRepo(db)
	coordRepo := repo.NewCoordinateRepo(db)
	entryRepo := repo.NewEntryRepo(db)
	volumeCalc := handler.NewMockVolumeCalculator()

	return &App{
		AuthMiddleware: authmw.AuthMiddleware(authRepo),
		AuthHandler:   authh.NewAuthHandler(authRepo),
		ScanHandler:   handler.NewScanHandler(scanRepo),
		CoordHandler:  handler.NewCoordinateHandler(coordRepo, scanRepo, entryRepo, volumeCalc),
		VolumeHandler: handler.NewVolumeHandler(scanRepo, entryRepo, volumeCalc),
	}
}
